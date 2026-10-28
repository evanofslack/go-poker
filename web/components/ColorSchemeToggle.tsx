import { ActionIcon, useComputedColorScheme, useMantineColorScheme } from "@mantine/core";
import { FiMoon, FiSun } from "react-icons/fi";

// ColorSchemeToggle switches between dark and light, starting from the system's; Mantine
// remembers the choice. Both icons
// render and CSS shows one, so the static page doesn't need to know the scheme.
export default function ColorSchemeToggle({ size = "sm" }: { size?: string }) {
    const { setColorScheme } = useMantineColorScheme();
    const colorScheme = useComputedColorScheme("dark");
    return (
        <ActionIcon
            size={size}
            title="light or dark"
            aria-label="light or dark"
            onClick={() => setColorScheme(colorScheme === "light" ? "dark" : "light")}
        >
            <FiSun className="hidden dark:block" />
            <FiMoon className="dark:hidden" />
        </ActionIcon>
    );
}
