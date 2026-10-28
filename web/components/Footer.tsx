import { FiGithub } from "react-icons/fi";
import ColorSchemeToggle from "./ColorSchemeToggle";

export default function Footer() {
  return (
    <footer className=" mt-10 flex w-screen flex-row items-center justify-between py-3 px-8 pb-4 text-sm font-light text-subtle md:mt-20">
      <p> &copy; 2026 go-poker </p>
      <div className="flex items-center gap-3">
        <ColorSchemeToggle size="md" />
        <a href="https://github.com/evanofslack/go-poker">
          <FiGithub size="1.2rem" />
        </a>
      </div>
    </footer>
  );
}
