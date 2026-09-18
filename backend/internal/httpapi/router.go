package httpapi

import "net/http"

// Modules bundles every module's handler group so NewRouter has one thing to
// take instead of a growing positional parameter list. Each field is
// optional (nil-safe) so a module can be wired in independently — useful in
// development when, say, GOOGLE_CLIENT_ID isn't set yet but you still want
// the rest of the API up.
type Modules struct {
	Auth      *AuthHandlers
	Bills     *BillHandlers
	Vault     *VaultHandlers
	Notes     *NoteHandlers
	Reminders *ReminderHandlers
	Events    *EventHandlers
	Documents *DocumentHandlers
	Training  *TrainingHandlers
	Diet      *DietHandlers
	Boards    *BoardHandlers
	Habits    *HabitHandlers
	Assistant *AssistantHandlers
	Telegram  *TelegramHandlers
	// MCP serves the Model Context Protocol; it checks its own token.
	MCP http.Handler
}

// NewRouter wires the API surface. It is intentionally reachable only from
// the Next.js server, never from a browser directly — see the top-level
// README's architecture section — so there is no CORS layer here: adding
// one would be defending against a request path that the deployment
// topology doesn't allow to exist.
//
// Routing is split across two muxes. Everything that needs a session is
// registered on an inner mux which is then mounted, wrapped in
// requireSession, under "/api/" on the outer one; the handful of public
// routes go straight on the outer mux. Wrapping each group individually was
// the alternative, but it puts the auth decision in fourteen places where
// forgetting one silently exposes a module — here the default is protected
// and each exception is visible in the few lines below. Go 1.22 patterns
// make this safe: the outer "/api/" is less specific than the exact public
// patterns, so those win without any ordering subtlety.
func NewRouter(h *Handlers, m Modules) http.Handler {
	mux := http.NewServeMux()
	api := http.NewServeMux()

	// Public: health is for the container probe, which has no credentials,
	// and login is how a session is obtained in the first place.
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if a := m.Auth; a != nil {
		mux.HandleFunc("POST /api/auth/login", a.Login)

		api.HandleFunc("POST /api/auth/logout", a.Logout)
		api.HandleFunc("GET /api/auth/me", a.Me)
		api.HandleFunc("POST /api/auth/password", a.ChangePassword)
		api.HandleFunc("POST /api/auth/verify-password", a.VerifyPassword)
	}

	api.HandleFunc("GET /api/categories", h.ListCategories)
	api.HandleFunc("POST /api/categories", h.CreateCategory)
	api.HandleFunc("PUT /api/categories/{id}", h.UpdateCategory)
	api.HandleFunc("DELETE /api/categories/{id}", h.DeleteCategory)
	api.HandleFunc("PATCH /api/categories/{id}/budget", h.UpdateCategoryBudget)
	api.HandleFunc("GET /api/credit-cards", h.ListCreditCards)
	api.HandleFunc("POST /api/credit-cards", h.CreateCreditCard)
	api.HandleFunc("PUT /api/credit-cards/{id}", h.UpdateCreditCard)
	api.HandleFunc("DELETE /api/credit-cards/{id}", h.DeleteCreditCard)
	api.HandleFunc("GET /api/months/{month}", h.MonthSummary)
	api.HandleFunc("POST /api/transactions", h.CreateTransaction)
	api.HandleFunc("PATCH /api/transactions/{id}", h.UpdateTransaction)
	api.HandleFunc("DELETE /api/transactions/{id}", h.DeleteTransaction)

	if b := m.Bills; b != nil {
		api.HandleFunc("GET /api/bills", b.List)
		api.HandleFunc("POST /api/bills", b.Create)
		api.HandleFunc("GET /api/bills/summary", b.Summary)
		api.HandleFunc("GET /api/bills/received", b.ReceivedTotal)
		api.HandleFunc("POST /api/bills/{id}/paid", b.MarkPaid)
		api.HandleFunc("POST /api/bills/{id}/pay", b.Pay)
		api.HandleFunc("DELETE /api/bills/{id}/paid", b.Unpay)
		api.HandleFunc("POST /api/bills/{id}/end-series", b.EndSeries)
		api.HandleFunc("DELETE /api/bills/{id}/end-series", b.ResumeSeries)
		api.HandleFunc("PUT /api/bills/{id}", b.Update)
		api.HandleFunc("DELETE /api/bills/{id}", b.Delete)
	}

	if v := m.Vault; v != nil {
		api.HandleFunc("GET /api/vault", v.List)
		api.HandleFunc("POST /api/vault", v.Create)
		api.HandleFunc("PUT /api/vault/{id}", v.Update)
		api.HandleFunc("DELETE /api/vault/{id}", v.Delete)
		api.HandleFunc("POST /api/vault/{id}/reveal", v.Reveal)
	}

	if n := m.Notes; n != nil {
		api.HandleFunc("GET /api/notes", n.List)
		api.HandleFunc("POST /api/notes", n.Create)
		api.HandleFunc("GET /api/notes/{id}", n.Get)
		api.HandleFunc("PUT /api/notes/{id}", n.Update)
		api.HandleFunc("DELETE /api/notes/{id}", n.Delete)
		api.HandleFunc("GET /api/note-categories", n.ListCategories)
		api.HandleFunc("POST /api/note-categories", n.CreateCategory)
		api.HandleFunc("PUT /api/note-categories/{id}", n.UpdateCategory)
		api.HandleFunc("DELETE /api/note-categories/{id}", n.DeleteCategory)
	}

	if rm := m.Reminders; rm != nil {
		api.HandleFunc("GET /api/reminders", rm.List)
		api.HandleFunc("POST /api/reminders", rm.Create)
		api.HandleFunc("PATCH /api/reminders/{id}", rm.SetDone)
		api.HandleFunc("PUT /api/reminders/{id}", rm.Update)
		api.HandleFunc("DELETE /api/reminders/{id}", rm.Delete)
	}

	if d := m.Documents; d != nil {
		api.HandleFunc("GET /api/document-folders", d.ListFolders)
		api.HandleFunc("POST /api/document-folders", d.CreateFolder)
		api.HandleFunc("PUT /api/document-folders/{id}", d.RenameFolder)
		api.HandleFunc("DELETE /api/document-folders/{id}", d.DeleteFolder)
		api.HandleFunc("GET /api/documents", d.List)
		api.HandleFunc("POST /api/documents", d.Upload)
		api.HandleFunc("GET /api/documents/count", d.Count)
		api.HandleFunc("GET /api/documents/{id}/content", d.Content)
		api.HandleFunc("PUT /api/documents/{id}", d.Move)
		api.HandleFunc("DELETE /api/documents/{id}", d.Delete)
	}

	if t := m.Training; t != nil {
		api.HandleFunc("GET /api/workouts", t.List)
		api.HandleFunc("POST /api/workouts", t.Create)
		api.HandleFunc("PUT /api/workouts/{id}", t.Update)
		api.HandleFunc("DELETE /api/workouts/{id}", t.Delete)
	}

	if dt := m.Diet; dt != nil {
		api.HandleFunc("GET /api/meals", dt.List)
		api.HandleFunc("POST /api/meals", dt.Create)
		api.HandleFunc("PUT /api/meals/{id}", dt.Update)
		api.HandleFunc("DELETE /api/meals/{id}", dt.Delete)
		api.HandleFunc("GET /api/diet/targets", dt.Targets)
		api.HandleFunc("PUT /api/diet/targets", dt.SetTargets)
	}

	if a := m.Assistant; a != nil {
		api.HandleFunc("GET /api/assistant/tools", a.ListTools)
		api.HandleFunc("POST /api/assistant/tools/{name}", a.CallTool)
		api.HandleFunc("GET /api/assistant/settings", a.Settings)
		api.HandleFunc("PUT /api/assistant/settings", a.UpdateSettings)
		api.HandleFunc("GET /api/assistant/conversations", a.Conversations)
		api.HandleFunc("GET /api/assistant/conversations/{id}", a.Conversation)
		api.HandleFunc("DELETE /api/assistant/conversations/{id}", a.DeleteConversation)
		api.HandleFunc("POST /api/assistant/record", a.Record)
		api.HandleFunc("POST /api/assistant/chat", a.Send)
		api.HandleFunc("POST /api/assistant/voice/transcribe", a.Transcribe)
		api.HandleFunc("POST /api/assistant/voice/speak", a.Speak)
	}
	if tg := m.Telegram; tg != nil {
		api.HandleFunc("GET /api/assistant/telegram/settings", tg.Settings)
		api.HandleFunc("PUT /api/assistant/telegram/settings", tg.UpdateSettings)
		api.HandleFunc("POST /api/assistant/telegram/pair", tg.Pair)
		api.HandleFunc("DELETE /api/assistant/telegram/pair", tg.Unpair)
		api.HandleFunc("POST /api/assistant/telegram/test", tg.Test)
	}
	// MCP lives outside /api and authenticates its own bearer token, so it
	// must not go behind requireSession: its clients hold an MCP token, not
	// a browser session.
	if m.MCP != nil {
		mux.Handle("/mcp", m.MCP)
	}

	if hb := m.Habits; hb != nil {
		api.HandleFunc("GET /api/habits", hb.List)
		api.HandleFunc("POST /api/habits", hb.Create)
		api.HandleFunc("PUT /api/habits/{id}", hb.Update)
		api.HandleFunc("DELETE /api/habits/{id}", hb.Delete)
		api.HandleFunc("PUT /api/habits/{id}/log", hb.SetLog)
	}

	if bd := m.Boards; bd != nil {
		api.HandleFunc("GET /api/boards", bd.List)
		api.HandleFunc("POST /api/boards", bd.Create)
		api.HandleFunc("GET /api/boards/{id}", bd.Get)
		api.HandleFunc("PATCH /api/boards/{id}", bd.Rename)
		api.HandleFunc("PUT /api/boards/{id}/scene", bd.SaveScene)
		api.HandleFunc("DELETE /api/boards/{id}", bd.Delete)
	}

	if e := m.Events; e != nil {
		api.HandleFunc("GET /api/events", e.ListRange)
		api.HandleFunc("POST /api/events", e.Create)
		api.HandleFunc("PUT /api/events/{id}", e.Update)
		api.HandleFunc("DELETE /api/events/{id}", e.Delete)
		api.HandleFunc("GET /api/google/status", e.GoogleStatus)
		api.HandleFunc("GET /api/google/oauth/start", e.GoogleAuthStart)
		api.HandleFunc("GET /api/google/oauth/callback", e.GoogleCallback)
		api.HandleFunc("POST /api/google/sync", e.GoogleSync)
	}

	// Without an auth module there is nobody to authenticate against, so the
	// API mounts unguarded. That is the shape unit tests and a pre-bootstrap
	// checkout run in; in production cmd/api always passes Auth.
	if m.Auth != nil {
		mux.Handle("/api/", requireSession(m.Auth.auth, api))
	} else {
		mux.Handle("/api/", api)
	}

	return withMiddleware(mux)
}
