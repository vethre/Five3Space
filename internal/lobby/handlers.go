package lobby

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"main/internal/data"
)

// NOTE: User struct now matches logic in pages.go, keep them synced or unified.
type User struct {
	ID           string
	Nickname     string
	Tag          string
	AvatarURL    string
	Exp          int
	MaxExp       int
	Medals       int
	Trophies     int
	Level        int
	Coins        int
	Status       string
	XPPercentage string
	Language     string
	NameColor    string
	BannerColor  string
	Inventory    []string
}

type GameMode struct {
	ID           string
	Title        string
	Subtitle     string
	IsLocked     bool
	IsConstruct  bool
	SafeGradient template.CSS
	BtnText      string
	StatusText   string
	URL          string
}

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
	Lang         string
	MedalDetails []data.Medal
	ShowRegister bool
	ActivePage   string
}

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
	default:
		if isConstruct {
			return "UNDER CONSTRUCTION", "UNAVAILABLE"
		}
		if isLocked {
			return "LOCKED", "SECURE"
		}
		return "DEPLOY", "LIVE"
	}
}

func NewHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLobby(w, r, store)
	}
}

func NewGameHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderGame(w, r, store)
	}
}

func renderGame(w http.ResponseWriter, r *http.Request, store *data.Store) {
	userID := "guest"
	if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
		userID = c.Value
	} else if q := r.URL.Query().Get("userID"); q != "" {
		userID = q
	}

	lang := normalizeLang(r.URL.Query().Get("lang"))
	if lang == "" {
		lang = "en"
	}

	data := struct {
		UserID string
		Lang   string
	}{UserID: userID, Lang: lang}

	tmplPath := filepath.Join("web", "templates", "game.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Could not load game", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func normalizeLang(raw string) string {
	switch raw {
	case "ua", "ru", "en":
		return raw
	default:
		return ""
	}
}

func renderLobby(w http.ResponseWriter, r *http.Request, store *data.Store) {
	requestedLang := normalizeLang(r.URL.Query().Get("lang"))

	var userID string
	hadCookie := false

	if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
		userID = c.Value
		hadCookie = true
	} else if q := r.URL.Query().Get("userID"); q != "" {
		userID = q
		hadCookie = true
	}

	var selected data.UserData
	userFound := false
	inv := []string{}
	if userID != "" {
		if u, ok := store.GetUser(userID); ok {
			selected = u
			userFound = true
			inv, _ = store.GetUserInventory(userID)
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

	if !userFound {
		hadCookie = false
		http.SetCookie(w, &http.Cookie{Name: "user_id", Value: "", Path: "/", MaxAge: -1})
	} else if userID != "" {
		http.SetCookie(w, &http.Cookie{Name: "user_id", Value: userID, Path: "/", MaxAge: 86400 * 30})
	}

	var user User
	if !userFound {
		user = User{
			Nickname: "Guest", Tag: "----", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=guest",
			MaxExp: 1, Status: "offline", Language: lang, NameColor: "white", BannerColor: "default",
		}
	} else {
		avatar := selected.CustomAvatar
		if avatar == "" {
			avatar = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", selected.Nickname)
		}

		user = User{
			ID: selected.ID, Nickname: selected.Nickname, Tag: fmt.Sprintf("%04d", selected.Tag),
			AvatarURL: avatar,
			Exp:       selected.Exp, MaxExp: selected.MaxExp, Medals: len(selected.Medals), Trophies: selected.Trophies,
			Level: selected.Level, Coins: selected.Coins, Status: selected.Status, Language: lang,
			NameColor: selected.NameColor, BannerColor: selected.BannerColor, Inventory: inv,
		}
	}

	pct := 0
	if user.MaxExp > 0 {
		pct = (user.Exp * 100) / user.MaxExp
	}
	user.XPPercentage = fmt.Sprintf("%d%%", pct)

	friendList, _ := store.ListFriends(user.ID)
	medalDetails := []data.Medal{}
	if userFound {
		medalDetails = store.MedalDetails(selected.Medals)
	}

	btn1, stat1 := getModeTexts(lang, false, false)

	modes := []GameMode{
		{
			ID: "chibiki", Title: "Chibiki Royale", Subtitle: "Clash Royale-style",
			SafeGradient: template.CSS("linear-gradient(135deg, #ff4e50 0%, #f9d423 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/game?mode=chibiki&lang=%s", lang),
		},
		{
			ID: "bobik", Title: "Bobik Shooter", Subtitle: "FPS-style",
			SafeGradient: template.CSS("linear-gradient(135deg, #36d1dc 0%, #5b86e5 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/bobik?lang=%s&userID=%s&nick=%s", lang, user.ID, user.Nickname),
		},
		{
			ID: "party", Title: "Five3Fun", Subtitle: "Party Game (2-8 Players)",
			SafeGradient: template.CSS("linear-gradient(135deg, #FF6B6B 0%, #4ECDC4 100%)"),
			BtnText:      "JOIN", StatusText: "OPEN", URL: fmt.Sprintf("/party?lang=%s&userID=%s", lang, user.ID),
			IsConstruct: false, IsLocked: false,
		},
	}

	pageData := PageData{
		User: user, Friends: friendList, Modes: modes, Text: t, Lang: lang,
		MedalDetails: medalDetails, ShowRegister: !hadCookie && !userFound, ActivePage: "lobby",
	}

	tmplPath := filepath.Join("web", "templates", "lobby.html")
	tmpl, _ := template.ParseFiles(tmplPath)
	tmpl.Execute(w, pageData)
}

func NewCustomizeSaveHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) Витягуємо userID з кукі або з query
		userID := ""
		if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
			userID = c.Value
		} else if q := r.URL.Query().Get("userID"); q != "" {
			userID = q
		}

		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2) Обмежуємо розмір JSON (аватарка + решта)
		r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)

		var req struct {
			NameColor    string `json:"name_color"`
			BannerColor  string `json:"banner_color"`
			CustomAvatar string `json:"custom_avatar"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Body too large or invalid", http.StatusBadRequest)
			return
		}

		// 3) Зберігаємо в БД
		if err := store.UpdateProfileLook(userID, req.NameColor, req.BannerColor, req.CustomAvatar); err != nil {
			http.Error(w, "Error saving", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
