package lobby

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"main/internal/data"
)

func getModeTexts(lang string, isLocked, isConstruct bool) (string, string) {
	switch lang {
	case "ua":
		if isConstruct {
			return "В Розробці", "НЕДОСТУПНО"
		}
		if isLocked {
			return "ЗАЧИНЕНО", "ЗАХИЩЕНО"
		}
		return "ГРАТИ", "ГОТОВО"
	case "ru":
		if isConstruct {
			return "В РАЗРАБОТКЕ", "НЕДОСТУПНО"
		}
		if isLocked {
			return "ЗАКРЫТО", "ЗАЩИЩЕНО"
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
			Nickname: "Guest", Tag: "----", AvatarURL: template.URL("https://api.dicebear.com/7.x/avataaars/svg?seed=guest"),
			MaxExp: 1, Status: "offline", Language: lang, NameColor: "white", BannerColor: "default",
		}
	} else {
		avatar := selected.CustomAvatar
		if avatar == "" {
			avatar = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", selected.Nickname)
		}

		user = User{
			ID: selected.ID, Nickname: selected.Nickname, Tag: fmt.Sprintf("%04d", selected.Tag),
			AvatarURL: template.URL(avatar),
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

	// UPDATED MODES TO USE TRANSLATIONS
	modes := []GameMode{
		{
			ID: "chibiki", Title: t.ChibikiTitle, Subtitle: t.ChibikiSub,
			SafeGradient: template.CSS("linear-gradient(135deg, #ff4e50 0%, #f9d423 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/game?mode=chibiki&lang=%s", lang),
		},
		{
			ID: "bobik", Title: t.BobikTitle, Subtitle: t.BobikSub,
			SafeGradient: template.CSS("linear-gradient(135deg, #36d1dc 0%, #5b86e5 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/bobik?lang=%s&userID=%s&nick=%s", lang, user.ID, user.Nickname),
		},
		{
			ID: "party", Title: t.PartyTitle, Subtitle: t.PartySub,
			SafeGradient: template.CSS("linear-gradient(135deg, #FF6B6B 0%, #4ECDC4 100%)"),
			BtnText:      "JOIN", StatusText: "OPEN", URL: fmt.Sprintf("/party?lang=%s&userID=%s", lang, user.ID),
			IsConstruct: false, IsLocked: false,
		},
		{
			ID: "slotix", Title: t.SlotixTitle, Subtitle: t.SlotixSub,
			SafeGradient: template.CSS("linear-gradient(135deg, #ffd700 0%, #ff6b00 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/slotix?lang=%s&userID=%s", lang, user.ID),
		},
		{
			ID: "upsidedown", Title: t.UpsideDownTitle, Subtitle: t.UpsideDownSub,
			SafeGradient: template.CSS("linear-gradient(135deg, #1a0000 0%, #4a0000 50%, #000000 100%)"),
			BtnText:      btn1, StatusText: stat1, URL: fmt.Sprintf("/upsidedown?lang=%s&userID=%s", lang, user.ID),
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

		if err := store.UpdateProfileLook(userID, req.NameColor, req.BannerColor, req.CustomAvatar); err != nil {
			http.Error(w, "Error saving", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func add(a, b int) int { return a + b }

func NewLeaderboardHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageData := commonPage(w, r, store)

		rawLeaders, err := store.GetLeaderboard()
		if err != nil {
			rawLeaders = []data.UserData{}
		}

		var displayLeaders []User
		for _, u := range rawLeaders {
			avatarSrc := u.CustomAvatar
			if avatarSrc == "" {
				avatarSrc = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", u.Nickname)
			}

			displayLeaders = append(displayLeaders, User{
				Nickname:  u.Nickname,
				Level:     u.Level,
				Trophies:  u.Trophies,
				NameColor: u.NameColor,
				AvatarURL: template.URL(avatarSrc),
			})
		}

		data := struct {
			User    User
			Lang    string
			Text    Translations
			Leaders []User
		}{
			User:    pageData.User,
			Lang:    pageData.Lang,
			Text:    pageData.Text, // Pass translations here!
			Leaders: displayLeaders,
		}

		tmplPath := filepath.Join("web", "templates", "leaderboard.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			http.Error(w, "Template Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}
}
