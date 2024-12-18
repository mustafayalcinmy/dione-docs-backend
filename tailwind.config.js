/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./**/*.{html,js}", "./**/*.templ","./**/*.go"],
    theme: {
        extend: {
            colors: {
                gruvbox: {
                    bg: "#282828", // Dark background
                    fg: "#ebdbb4", // Light foreground
                    red: "#cc241d",
                    green: "#98971a",
                    yellow: "#d79921",
                    blue: "#458588",
                    purple: "#b16286",
                    aqua: "#689d6a",
                    orange: "#d65d0e",
                },
            },
        },
    },
    plugins: [],
};
