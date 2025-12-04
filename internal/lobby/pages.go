package lobby

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"main/internal/data"
)

// commonPage builds the shared data model for all pages.
func commonPage(w http.ResponseWriter, r *http.Request, store *data.Store) PageData {
	requestedLang := normalizeLang(r.URL.Query().Get("lang"))

	userID := r.URL.Query().Get("userID")
	hadCookie := false
	if userID != "" {
		hadCookie = true
	} else if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
		userID = c.Value
		hadCookie = true
	}

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

	if !userFound {
		hadCookie = false
		http.SetCookie(w, &http.Cookie{
			Name:   "user_id",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

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

	return PageData{
		User:         user,
		Friends:      friendList,
		Text:         t,
		Lang:         lang,
		MedalDetails: medalDetails,
		ShowRegister: !hadCookie,
	}
}

func NewFriendsHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := commonPage(w, r, store)
		data.ActivePage = "friends"
		tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "friends.html"))
		if err != nil {
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		_ = tmpl.Execute(w, data)
	}
}

func NewShopHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := commonPage(w, r, store)
		data.ActivePage = "shop"
		tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "shop.html"))
		if err != nil {
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		_ = tmpl.Execute(w, data)
	}
}

func NewCustomizeHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := commonPage(w, r, store)
		data.ActivePage = "customize"
		tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "customize.html"))
		if err != nil {
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		_ = tmpl.Execute(w, data)
	}
}
