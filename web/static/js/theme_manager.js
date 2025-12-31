class ThemeManager {
    constructor() {
        this.themes = ['default', 'upside-down', 'new-year'];
        this.currentTheme = localStorage.getItem('app-theme') || 'default';
        this.applyTheme(this.currentTheme);
    }

    setTheme(themeName) {
        if (!this.themes.includes(themeName)) return;
        this.currentTheme = themeName;
        localStorage.setItem('app-theme', themeName);
        this.applyTheme(themeName);
    }

    applyTheme(themeName) {
        document.body.classList.remove('upside-down-mode', 'new-year-mode');

        if (themeName === 'upside-down') {
            document.body.classList.add('upside-down-mode');
        } else if (themeName === 'new-year') {
            document.body.classList.add('new-year-mode');
        }
    }
}

window.themeManager = new ThemeManager();
