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
	Tag          string
	AvatarURL    string
	Exp          int
	MaxExp       int
	Medals       int
	Level        int
	Coins        int
	Status       string
	XPPercentage string
	Language     string
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
	Shop        string
	FriendsNav  string
	Customize   string
	Market      string
	MarketSub   string
	Hangar      string
	HangarSub   string
	LangCurrent string
}

type PageData struct {
	User         User
	Friends      []data.Friend
	Modes        []GameMode
	Text         Translations
	Lang         string // "en", "ua", "ru"
	MedalDetails []data.Medal
	ShowRegister bool
	ActivePage   string
}

func normalizeLang(raw string) string {
	switch raw {
	case "ua", "ru", "en":
		return raw
	default:
		return ""
	}
}

// --- Localization Data ---

var texts = map[string]Translations{
	"en": {
		LobbyName:   "FIVE3 Game Space",
		Level:       "LEVEL",
		XP:          "XP",
		DeployZone:  "DEPLOYMENT ZONE",
		Shop:        "Shop",
		FriendsNav:  "Friends",
		Customize:   "Customization",
		Market:      "Market",
		MarketSub:   "Supplies",
		Hangar:      "Hangar",
		HangarSub:   "Style",
		LangCurrent: "ENG",
	},
	"ua": {
		LobbyName:   "П'ЯТЬ3 Ігро-Space",
		Level:       "Рівень",
		XP:          "Досвід",
		DeployZone:  "Зона висадки",
		Shop:        "Ринок",
		FriendsNav:  "Друзі",
		Customize:   "Налаштування",
		Market:      "Ринок",
		MarketSub:   "Постачання",
		Hangar:      "Ангар",
		HangarSub:   "Стиль",
		LangCurrent: "UKR",
	},
	"ru": {
		LobbyName:   "ПЯТЬ3 Игро-Space",
		Level:       "Уровень",
		XP:          "Опыт",
		DeployZone:  "Зона высадки",
		Shop:        "Рынок",
		FriendsNav:  "Друзья",
		Customize:   "Настройки",
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
	// 1. Language
	requestedLang := normalizeLang(r.URL.Query().Get("lang"))

	// 2. Пытаемся достать userID
	var userID string
	hadCookie := false

	// сначала query-параметр
	if q := r.URL.Query().Get("userID"); q != "" {
		userID = q
		hadCookie = true
	} else if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
		// потом кука
		userID = c.Value
		hadCookie = true
	}

	// 3. Пытаемся найти юзера в базе
	var selected data.UserData
	userFound := false
	if userID != "" {
		if u, ok := store.GetUser(userID); ok {
			selected = u
			userFound = true
		}
	}

	lang := requestedLang
	if lang == "" && selected.Language != "" {
		lang = normalizeLang(selected.Language)
	}
	if lang == "" {
		lang = "en"
	}
	t := texts[lang]

	// 4. Если юзер не найден — сбрасываем куку и включаем регистрацию
	if !userFound {
		hadCookie = false
		http.SetCookie(w, &http.Cookie{
			Name:   "user_id",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	// 5. Маппим в view-модель
	var user User
	if !userFound {
		user = User{
			ID:        "",
			Nickname:  "ERROR LOADING",
			Tag:       "----",
			AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=error&backgroundColor=ffdfbf",
			Medals:    0,
			Level:     0,
			Exp:       0,
			MaxExp:    1,
			Coins:     0,
			Status:    "offline",
			Language:  lang,
		}
	} else {
		user = User{
			ID:        selected.ID,
			Nickname:  selected.Nickname,
			Tag:       fmt.Sprintf("%04d", selected.Tag),
			AvatarURL: fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", selected.Nickname),
			Exp:       selected.Exp,
			MaxExp:    selected.MaxExp,
			Medals:    len(selected.Medals),
			Level:     selected.Level,
			Coins:     selected.Coins,
			Status:    selected.Status,
			Language:  lang,
		}
	}

	// 6. XP %
	pct := 0
	if user.MaxExp > 0 {
		pct = (user.Exp * 100) / user.MaxExp
	}
	user.XPPercentage = fmt.Sprintf("%d%%", pct)

	// 7. Друзья и медали
	friendList, _ := store.ListFriends(user.ID)
	medalDetails := []data.Medal{}
	if userFound {
		medalDetails = store.MedalDetails(selected.Medals)
	}

	// 8. Game modes
	btn1, stat1 := getModeTexts(lang, false, false)
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
			IsConstruct:  false,
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
			BtnText:      btn3,
			StatusText:   stat3,
		},
	}

	data := PageData{
		User:         user,
		Friends:      friendList,
		Modes:        modes,
		Text:         t,
		Lang:         lang,
		MedalDetails: medalDetails,
		ShowRegister: !hadCookie,
		ActivePage:   "lobby",
	}

	tmplPath := filepath.Join("web", "templates", "lobby.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Error parsing: %v", err)
		http.Error(w, "Could not load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Execution error: %v", err)
	}
}
