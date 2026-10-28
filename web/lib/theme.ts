import { ActionIcon, Button, createTheme, CSSVariablesResolver, MantineColorsTuple } from "@mantine/core";

// The palette lives here; the role tokens in styles/index.css point at these colors.
// Shades are Tailwind's, in hex so Mantine can work out contrast.

// dark is zinc, in the slots Mantine gives each role: 0 text, 2 dimmed, 4 borders,
// 5 hover, 6 default buttons and inputs, 7 the page, 8 panels
const dark: MantineColorsTuple = [
    "#f4f4f5",
    "#d4d4d8",
    "#a1a1aa",
    "#71717a",
    "#52525b",
    "#48484f", // between 600 and 700, so default buttons have a hover
    "#3f3f46",
    "#27272a",
    "#18181b",
    "#09090b",
];

// gray is zinc too, lightest first, for gray buttons and icons
const gray: MantineColorsTuple = [
    "#fafafa",
    "#f4f4f5",
    "#e4e4e7",
    "#d4d4d8",
    "#a1a1aa",
    "#71717a",
    "#52525b",
    "#3f3f46",
    "#27272a",
    "#18181b",
];

const cyan: MantineColorsTuple = [
    "#ecfeff",
    "#cffafe",
    "#a5f3fc",
    "#67e8f9",
    "#22d3ee",
    "#06b6d4",
    "#0891b2",
    "#0e7490",
    "#155e75",
    "#164e63",
];

// rose, standing in for red
const red: MantineColorsTuple = [
    "#fff1f2",
    "#ffe4e6",
    "#fecdd3",
    "#fda4af",
    "#fb7185",
    "#f43f5e",
    "#e11d48",
    "#be123c",
    "#9f1239",
    "#881337",
];

// bar sizes the buttons in the footer: size md on a phone, bigger from sm and on tall screens
export const bar = "px-3 sm:h-12 sm:px-5 sm:text-xl tall:h-12";

export const theme = createTheme({
    colors: { dark, gray, cyan, red },
    primaryColor: "cyan",
    primaryShade: { light: 6, dark: 5 },
    autoContrast: true,
    fontFamily:
        'ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"',
    lineHeights: { md: "1.5" },
    defaultRadius: "md",
    radius: { md: "0.375rem" },
    components: {
        Button: Button.extend({
            defaultProps: { variant: "default", size: "sm" },
            // subtle buttons back out of things, so they're gray unless given a color
            vars: (_, props) =>
                props.variant === "subtle" && !props.color
                    ? {
                          root: {
                              "--button-color": "var(--mantine-color-gray-light-color)",
                              "--button-hover": "var(--mantine-color-gray-light-hover)",
                          },
                      }
                    : { root: {} },
        }),
        ActionIcon: ActionIcon.extend({ defaultProps: { variant: "subtle", color: "gray" } }),
    },
});

// Mantine's dark scheme takes outline and light text colors 4-5 shades below primaryShade,
// which suits its own palettes but lands near white in Tailwind's. Use the 400s instead.
export const resolver: CSSVariablesResolver = () => ({
    variables: {},
    // default buttons filled, as they are in dark, so they stand off the page
    light: {
        "--mantine-color-default": "var(--mantine-color-gray-2)",
        "--mantine-color-default-hover": "var(--mantine-color-gray-3)",
    },
    dark: Object.fromEntries(
        ["gray", "cyan", "red"].flatMap((c) => [
            [`--mantine-color-${c}-outline`, `var(--mantine-color-${c}-4)`],
            [`--mantine-color-${c}-light-color`, `var(--mantine-color-${c}-4)`],
        ])
    ),
});
