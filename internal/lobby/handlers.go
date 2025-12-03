package lobby

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"main/internal/data"
)

// --- Data Structures ---

type User struct {
	ID           string
	Nickname     string
	AvatarURL    string
	Exp          int
	MaxExp       int
	Medals       int
	Level        int
	XPPercentage string
}

type GameMode struct {
	ID           string
	Title        string
	Subtitle     string
	IsLocked     bool
	IsConstruct  bool
	SafeGradient template.CSS
	BtnText      string // Localized button text
	StatusText   string // Localized status text
	URL          string
}

// Translations holds all text for the UI
type Translations struct {
	LobbyName   string
	Level       string
	XP          string
	DeployZone  string
	Market      string
	MarketSub   string
	Hangar      string
	HangarSub   string
	LangCurrent string
}

type PageData struct {
	User         User
	Modes        []GameMode
	Text         Translations
	Lang         string // "en", "ua", "ru"
	MedalDetails []data.Medal
}

// --- Localization Data ---

var texts = map[string]Translations{
	"en": {
		LobbyName:   "ANTEIKU LOBBY",
		Level:       "LEVEL",
		XP:          "XP",
		DeployZone:  "DEPLOYMENT ZONE",
		Market:      "Market",
		MarketSub:   "Supplies",
		Hangar:      "Hangar",
		HangarSub:   "Style",
		LangCurrent: "ENG",
	},
	"ua": {
		LobbyName:   "ANTEIKU LOBBY",
		Level:       "Рівень",
		XP:          "Досвід",
		DeployZone:  "Зона висадки",
		Market:      "Ринок",
		MarketSub:   "Постачання",
		Hangar:      "Ангар",
		HangarSub:   "Стиль",
		LangCurrent: "UKR",
	},
	"ru": {
		LobbyName:   "ANTEIKU LOBBY",
		Level:       "Уровень",
		XP:          "Опыт",
		DeployZone:  "Зона высадки",
		Market:      "Рынок",
		MarketSub:   "Поставки",
		Hangar:      "Ангар",
		HangarSub:   "Стиль",
		LangCurrent: "RUS",
	},
}

// Helper to get button text based on lang
func getModeTexts(lang string, isLocked, isConstruct bool) (string, string) {
	switch lang {
	case "ua":
		if isConstruct {
			return "В Розробці", "НЕДОСТУПНО"
		}
		if isLocked {
			return "ЗАЧИНЕНО", "ОХОРОНА"
		}
		return "ГРАТИ", "ГОТОВО"
	case "ru":
		if isConstruct {
			return "В РАЗРАБОТКЕ", "НЕДОСТУПНО"
		}
		if isLocked {
			return "ЗАКРЫТО", "ОХРАНА"
		}
		return "ИГРАТЬ", "ГОТОВО"
	default: // en
		if isConstruct {
			return "UNDER CONSTRUCTION", "UNAVAILABLE"
		}
		if isLocked {
			return "LOCKED", "SECURE"
		}
		return "DEPLOY", "LIVE"
	}
}

// --- Handler ---

func NewHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLobby(w, r, store)
	}
}

func renderLobby(w http.ResponseWriter, r *http.Request, store *data.Store) {
	// 1. Language Selection (Query param > default "en")
	lang := r.URL.Query().Get("lang")
	if lang != "ua" && lang != "ru" {
		lang = "en"
	}

	t := texts[lang]

	// 2. Load User Data from Store
	userID := r.URL.Query().Get("userID")
	var selected data.UserData
	var ok bool
	if userID != "" {
		selected, ok = store.GetUser(userID)
	}
	if !ok {
		if u, found := store.FirstUser(); found {
			selected = u
			ok = true
		}
	}

	var user User
	if !ok {
		log.Printf("No user data available")
		// Fallback to mock data
		user = User{
			ID:        "guest",
			Nickname:  "Error Loading",
			AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=error&backgroundColor=ffdfbf",
			Medals:    0,
		}
	} else {
		user = User{
			ID:        selected.ID,
			Nickname:  selected.Nickname,
			AvatarURL: fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", selected.Nickname),
			Exp:       selected.Exp,
			MaxExp:    selected.MaxExp,
			Medals:    len(selected.Medals),
			Level:     selected.Level,
		}
	}

	// Calculate XP logic in Go
	pct := 0
	if user.MaxExp > 0 {
		pct = (user.Exp * 100) / user.MaxExp
	}
	user.XPPercentage = fmt.Sprintf("%d%%", pct)

	// 3. Prepare Game Modes with Localization
	btn1, stat1 := getModeTexts(lang, false, false) // Chibiki is now playable
	btn2, stat2 := getModeTexts(lang, false, true)
	btn3, stat3 := getModeTexts(lang, true, false)

	chibikiURL := ""
	if user.ID != "" {
		chibikiURL = fmt.Sprintf("/game?mode=chibiki&userID=%s&lang=%s", user.ID, lang)
	}

	modes := []GameMode{
		{
			ID:           "chibiki",
			Title:        "Chibiki Royale",
			Subtitle:     "Clash Royale-style",
			IsConstruct:  false, // No longer under construction
			SafeGradient: template.CSS("linear-gradient(135deg, #ff4e50 0%, #f9d423 100%)"),
			BtnText:      btn1,
			StatusText:   stat1,
			URL:          chibikiURL,
		},
		{
			ID:           "bobik",
			Title:        "Bobik Shooter",
			Subtitle:     "FPS-style",
			IsConstruct:  true,
			SafeGradient: template.CSS("linear-gradient(135deg, #FF416C 0%, #FF4B2B 100%)"),
			BtnText:      stat2,
			StatusText:   btn2,
		},
		{
			ID:           "tba",
			Title:        "???",
			Subtitle:     "Top Secret",
			IsLocked:     true,
			SafeGradient: template.CSS("linear-gradient(135deg, #232526 0%, #414345 100%)"),
			BtnText:      btn3, // "Locked"
			StatusText:   stat3,
		},
	}

	data := PageData{
		User:         user,
		Modes:        modes,
		Text:         t,
		Lang:         lang,
		MedalDetails: store.MedalDetails(selected.Medals),
	}

	// 4. Parse & Execute
	tmplPath := filepath.Join("web", "templates", "lobby.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Error parsing: %v", err)
		http.Error(w, "Could not load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Execution error: %v", err)
	}
}
