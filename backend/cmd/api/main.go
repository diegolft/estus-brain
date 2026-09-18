// Command api is the Estus Brain backend: a single Go binary that owns the
// ledger, applies its own migrations on boot, and serves a small JSON API
// consumed exclusively by the Next.js frontend running alongside it.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Bundled so the owner's time zone resolves even on a minimal container.
	_ "time/tzdata"

	"github.com/rafael/estus-vault/backend/internal/assistant"
	"github.com/rafael/estus-vault/backend/internal/config"
	"github.com/rafael/estus-vault/backend/internal/domain"
	"github.com/rafael/estus-vault/backend/internal/httpapi"
	"github.com/rafael/estus-vault/backend/internal/service"
	"github.com/rafael/estus-vault/backend/internal/store/postgres"
	"github.com/rafael/estus-vault/backend/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(ctx, "migrations"); err != nil {
		return err
	}

	// Auth must come up before anything that depends on a session. The
	// bootstrap account is what keeps a fresh deployment from being locked
	// out of an app that now demands a login.
	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	authService := service.NewAuthService(userRepo, sessionRepo)
	if err := authService.EnsureBootstrapUser(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return fmt.Errorf("bootstrap user: %w", err)
	}
	authHandlers := httpapi.NewAuthHandlers(authService)

	// Expired sessions are already ignored by every lookup, so this sweep is
	// housekeeping rather than a security control — a failure is worth a log
	// line and nothing more.
	if n, err := authService.SweepExpiredSessions(ctx); err != nil {
		slog.Warn("expired session sweep failed", "error", err)
	} else if n > 0 {
		slog.Info("expired sessions removed", "count", n)
	}

	categoryRepo := postgres.NewCategoryRepo(db)
	cardRepo := postgres.NewCreditCardRepo(db)
	transactionRepo := postgres.NewTransactionRepo(db)

	transactionService := service.NewTransactionService(transactionRepo, categoryRepo, cardRepo)
	dashboardService := service.NewDashboardService(transactionRepo)

	handlers := httpapi.NewHandlers(categoryRepo, cardRepo, transactionService, dashboardService)

	billRepo := postgres.NewBillRepo(db)
	billService := service.NewBillService(billRepo, categoryRepo, cardRepo)
	billHandlers := httpapi.NewBillHandlers(billService)

	noteRepo := postgres.NewNoteRepo(db)
	noteService := service.NewNoteService(noteRepo)
	noteCategoryRepo := postgres.NewNoteCategoryRepo(db)
	noteCategoryService := service.NewNoteCategoryService(noteCategoryRepo)
	noteHandlers := httpapi.NewNoteHandlers(noteService, noteCategoryService)

	reminderRepo := postgres.NewReminderRepo(db)
	reminderService := service.NewReminderService(reminderRepo)
	reminderHandlers := httpapi.NewReminderHandlers(reminderService)

	eventRepo := postgres.NewEventRepo(db)
	googleTokenRepo := postgres.NewGoogleTokenRepo(db)
	googleService := service.NewGoogleService(googleTokenRepo)
	eventService := service.NewEventService(eventRepo, googleService)
	eventHandlers := httpapi.NewEventHandlers(eventService, googleService)

	trainingService := service.NewTrainingService(postgres.NewWorkoutRepo(db))
	trainingHandlers := httpapi.NewTrainingHandlers(trainingService)

	dietService := service.NewDietService(postgres.NewMealRepo(db))
	dietHandlers := httpapi.NewDietHandlers(dietService)

	habitService := service.NewHabitService(postgres.NewHabitRepo(db))
	habitHandlers := httpapi.NewHabitHandlers(habitService)

	boardService := service.NewBoardService(postgres.NewBoardRepo(db))
	boardHandlers := httpapi.NewBoardHandlers(boardService)

	// Documents keep their bytes on disk; if that directory can't be opened
	// the module switches off rather than taking the whole API down with it.
	var documentHandlers *httpapi.DocumentHandlers
	documentFolderRepo := postgres.NewDocumentFolderRepo(db)
	documentRepo := postgres.NewDocumentRepo(db)
	documentService, documentErr := service.NewDocumentService(documentFolderRepo, documentRepo, cfg.DocumentsDir)
	if documentErr != nil {
		slog.Warn("documents module disabled: storage directory unavailable", "dir", cfg.DocumentsDir, "error", documentErr)
	} else {
		slog.Info("documents storage ready", "dir", documentService.Root())
		documentHandlers = httpapi.NewDocumentHandlers(documentService)
	}

	// The vault module needs VAULT_ENCRYPTION_KEY to exist at all — treat its
	// absence as "module disabled" rather than a fatal boot error, so the
	// rest of the app still comes up on a fresh checkout before it's
	// configured. Access to /senhas is now the app's own session, and Reveal
	// re-confirms the account password on top of it.
	var vaultHandlers *httpapi.VaultHandlers
	vaultRepo := postgres.NewVaultRepo(db)
	vaultService, vaultErr := service.NewVaultService(vaultRepo)
	if vaultErr != nil {
		slog.Warn("vault module disabled: set VAULT_ENCRYPTION_KEY to enable /senhas", "vault_error", vaultErr)
	} else {
		vaultHandlers = httpapi.NewVaultHandlers(vaultService, authService)
	}

	// The assistant: every action as a tool, shared by the chat, MCP and the
	// AI engines. The vault stays out of it.
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("ASSISTANT_TZ: %w", err)
	}
	tools := assistant.New(assistant.Deps{
		Categories:     categoryRepo,
		Cards:          cardRepo,
		Transactions:   transactionService,
		TransactionLog: transactionRepo,
		Dashboard:      dashboardService,
		Bills:          billService,
		Notes:          noteService,
		NoteCategories: noteCategoryService,
		Reminders:      reminderService,
		Events:         eventService,
		Habits:         habitService,
		Training:       trainingService,
		Diet:           dietService,
		Documents:      documentServiceOrNil(documentService, documentErr),
		Boards:         boardService,
		Location:       location,
	})
	mcpToken, err := assistant.LoadOrCreateToken(cfg.AssistantDir, cfg.MCPToken)
	if err != nil {
		return fmt.Errorf("assistant token: %w", err)
	}
	chatCfg := assistant.ChatConfig{
		Dir:            cfg.AssistantDir,
		MCPURL:         "http://127.0.0.1:" + cfg.Port + "/mcp",
		MCPToken:       mcpToken,
		ToolsURL:       "http://127.0.0.1:" + cfg.Port + "/api/assistant/tools",
		MultiUser:      cfg.MultiUser,
		AppleBridgeBin: getenv("APPLE_BRIDGE_BIN", "../apple-bridge/.build/release/estus-apple-bridge"),
		AppleBridgeURL: getenv("APPLE_BRIDGE_URL", "http://127.0.0.1:8765"),
		Voice:          getenv("ASSISTANT_VOICE", "Luciana"),
	}
	if key, err := domain.LoadEncryptionKeyFromEnv(); err == nil {
		chatCfg.VaultKey = &key
	}
	chat := assistant.NewChat(tools, postgres.NewAssistantRepo(db), chatCfg)
	defer chat.Close()
	assistantHandlers := httpapi.NewAssistantHandlers(tools, chat)
	mcpHandler := tools.MCPHandler(func() []string { return []string{mcpToken} })
	slog.Info("assistant ready", "tools", len(tools.Tools()), "mcp", "/mcp")

	// Telegram: the owner's chat with the assistant, plus daily reports and
	// alerts. It long-polls, so it needs no public address.
	telegramBot := telegram.New(telegram.Config{
		Store: postgres.NewTelegramRepo(db),
		Chat:  chat,
		Data: telegram.ServiceData{
			ReminderSvc:     reminderService,
			EventSvc:        eventService,
			BillSvc:         billService,
			TrainingSvc:     trainingService,
			HabitSvc:        habitService,
			TransactionRepo: transactionRepo,
		},
		Voice:    chat.Voice(),
		Tools:    tools,
		Location: location,
		VaultKey: chatCfg.VaultKey,
		EnvToken: cfg.TelegramBotToken,
	})
	botDone := make(chan struct{})
	go func() {
		defer close(botDone)
		telegramBot.Run(ctx)
	}()

	// Keep the sessions table from accumulating dead rows on a process that
	// stays up for months. It ends with ctx, alongside the bot.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := authService.SweepExpiredSessions(ctx); err != nil {
					slog.Warn("expired session sweep failed", "error", err)
				} else if n > 0 {
					slog.Info("expired sessions removed", "count", n)
				}
			}
		}
	}()

	router := httpapi.NewRouter(handlers, httpapi.Modules{
		Auth:      authHandlers,
		Bills:     billHandlers,
		Vault:     vaultHandlers,
		Notes:     noteHandlers,
		Reminders: reminderHandlers,
		Events:    eventHandlers,
		Documents: documentHandlers,
		Training:  trainingHandlers,
		Diet:      dietHandlers,
		Boards:    boardHandlers,
		Habits:    habitHandlers,
		Assistant: assistantHandlers,
		Telegram:  httpapi.NewTelegramHandlers(telegramBot),
		MCP:       mcpHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("estus vault api listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	var serveErr error
	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		serveErr = srv.Shutdown(shutdownCtx)
	case serveErr = <-errCh:
	}
	// Let the bot finish its last writes before the deferred closes take the
	// database and the AI engines away from it.
	cancel()
	select {
	case <-botDone:
		slog.Info("telegram bot stopped")
	case <-time.After(10 * time.Second):
		slog.Warn("telegram bot did not stop in time")
	}
	return serveErr
}

// documentServiceOrNil keeps a disabled documents module out of the assistant.
func documentServiceOrNil(s *service.DocumentService, err error) *service.DocumentService {
	if err != nil {
		return nil
	}
	return s
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
