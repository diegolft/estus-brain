import { IconBrain } from "@/components/icons";
import { LoginForm } from "./LoginForm";
import "../ui.css";
import "./login.css";

export const metadata = {
  title: "Entrar — Estus Brain",
};

// The only page the proxy lets through without a session. ?next carries
// wherever the viewer was actually headed when they were bounced here, so
// signing in puts them there instead of at the núcleo.
export default async function LoginPage(props: PageProps<"/login">) {
  const { next } = await props.searchParams;
  const target = typeof next === "string" && next.startsWith("/") && !next.startsWith("//") ? next : "/";

  return (
    <div className="login-screen">
      <div className="login-card">
        <div className="login-brand">
          <span className="brand-mark">
            <IconBrain />
          </span>
          <h1>Estus Brain</h1>
        </div>

        <div className="panel">
          <p className="login-lede">Entre para abrir seu cérebro.</p>
          <LoginForm next={target} />
        </div>
      </div>
    </div>
  );
}
