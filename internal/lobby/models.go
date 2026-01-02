package lobby

import (
	"html/template"
	"main/internal/data"
)

type User struct {
	ID           string
	Nickname     string
	Tag          string
	AvatarURL    template.URL
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
	// General / Nav
	LobbyName   string
	Level       string
	XP          string
	DeployZone  string
	Shop        string
	FriendsNav  string
	Customize   string
	Leaderboard string
	Back        string
	Settings    string
	Menu        string
	Close       string
	Cancel      string
	Save        string
	Later       string
	Copy        string
	LangCurrent string

	// Auth & Status
	Login         string
	Register      string
	Logout        string
	Nickname      string
	Password      string
	Tag           string
	RememberMe    string
	Online        string
	Away          string
	Offline       string
	CreateProfile string
	LoginTitle    string

	// Settings Modal
	LangTitle    string
	StatusTitle  string
	AccountTitle string
	LangNote     string
	StatusNote   string
	AccountNote  string

	// Friends Page
	AddFriendBtn    string
	NoFriendsTitle  string
	NoFriendsDesc   string
	ChatAction      string
	RemoveAction    string
	AddFriendHeader string
	SendRequest     string
	ChatTitle       string

	// Customize Page
	CustomizeTitle   string
	NameColorTitle   string
	BannerTitle      string
	PreviewLabel     string
	ColorWhite       string
	ColorGold        string
	ColorRainbow     string
	ColorHawkins     string
	BannerDefault    string
	BannerGold       string
	BannerCyber      string
	BannerUpsideDown string

	// Shop Page
	ExclusiveStyles string
	Resources       string
	ItemRainbowName string
	ItemRainbowDesc string
	ItemGoldName    string
	ItemGoldDesc    string
	ItemCyberBanner string
	ItemCyberDesc   string
	ItemGoldBanner  string
	ItemGoldBannerD string
	ItemSack        string
	ItemChest       string

	// Leaderboard Page
	LeaderboardTitle string
	CurrentSeason    string
	SeasonName       string
	EndsIn           string
	Rank             string
	Trophies         string

	// Game Modes
	ChibikiTitle    string
	ChibikiSub      string
	BobikTitle      string
	BobikSub        string
	PartyTitle      string
	PartySub        string
	SlotixTitle     string
	SlotixSub       string
	UpsideDownTitle string
	UpsideDownSub   string

	// Express Game
	ExpressTitle       string
	ExpressSub         string
	ExpressScore       string
	ExpressGameOver    string
	ExpressFinalScore  string
	ExpressPlayAgain   string
	ExpressExit        string
	ExpressDoubleClash string
	ExpressTripleClash string
	ExpressQuadClash   string
	ExpressMegaClash   string

	// Cozy Fishing Case
	FishingTitle     string
	FishingSub       string
	FishingCast      string
	FishingReel      string
	FishingCatch     string
	FishingMiss      string
	FishingWait      string
	FishingScore     string
	FishingGameOver  string
	FishingPlayAgain string
	FishingExit      string
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
		Leaderboard: "Leaderboard",
		Back:        "Back",
		Settings:    "Settings",
		Menu:        "Menu",
		Close:       "Close",
		Cancel:      "Cancel",
		Save:        "Save",
		Later:       "Later",
		Copy:        "Copy",
		LangCurrent: "ENG",

		Login:         "Login",
		Register:      "Register",
		Logout:        "Log Out",
		Nickname:      "Nickname",
		Password:      "Password",
		Tag:           "Tag (e.g. 0001)",
		RememberMe:    "Remember me on this device",
		Online:        "Online",
		Away:          "Away",
		Offline:       "Offline",
		CreateProfile: "Create your profile",
		LoginTitle:    "Login",

		LangTitle:    "Language",
		StatusTitle:  "Status",
		AccountTitle: "Account",
		LangNote:     "We save this to your account so every page uses it.",
		StatusNote:   "Tip: click the status dot on your avatar to toggle quickly.",
		AccountNote:  "Remember me keeps you signed in on this device.",

		AddFriendBtn:    "+ Add Friend",
		NoFriendsTitle:  "No friends yet",
		NoFriendsDesc:   "Add friends using their Nickname and Tag #.",
		ChatAction:      "Chat",
		RemoveAction:    "Remove",
		AddFriendHeader: "Add Friend",
		SendRequest:     "Send Request",
		ChatTitle:       "Chat",

		CustomizeTitle:   "Customize",
		NameColorTitle:   "Name Color",
		BannerTitle:      "Lobby Banner",
		PreviewLabel:     "PREVIEW",
		ColorWhite:       "Standard White",
		ColorGold:        "Gold",
		ColorRainbow:     "Rainbow",
		ColorHawkins:     "Hawkins Lights",
		BannerDefault:    "Default Dark",
		BannerGold:       "Golden Glow",
		BannerCyber:      "Cyber Punk",
		BannerUpsideDown: "The Upside Down",

		ExclusiveStyles: "Exclusive Styles",
		Resources:       "Resources",
		ItemRainbowName: "Rainbow Name",
		ItemRainbowDesc: "Make your nickname shimmer with all colors.",
		ItemGoldName:    "Gold Name",
		ItemGoldDesc:    "A prestigious golden glow for your name.",
		ItemCyberBanner: "Cyber Banner",
		ItemCyberDesc:   "Futuristic banner for your lobby background.",
		ItemGoldBanner:  "Gold Banner",
		ItemGoldBannerD: "Show off your wealth with this banner.",
		ItemSack:        "Sack of Coins",
		ItemChest:       "Chest of Coins",

		LeaderboardTitle: "Leaderboard",
		CurrentSeason:    "Current Season",
		SeasonName:       "SEASON 2: WINTER THINGS",
		EndsIn:           "Ends in",
		Rank:             "Rank",
		Trophies:         "Trophies",

		ChibikiTitle:    "Chibiki Royale",
		ChibikiSub:      "Clash Royale-style",
		BobikTitle:      "Bobik Shooter",
		BobikSub:        "FPS-style",
		PartyTitle:      "Five3Fun",
		PartySub:        "Party Game (2-8 Players)",
		SlotixTitle:     "Slotix",
		SlotixSub:       "Lucky Slot Machine",
		UpsideDownTitle: "The Upside Down",
		UpsideDownSub:   "Stranger Things Survival",

		ExpressTitle:       "New Year's Express",
		ExpressSub:         "Festive Block Puzzle",
		ExpressScore:       "Score",
		ExpressGameOver:    "Game Over!",
		ExpressFinalScore:  "Your Final Score: ",
		ExpressPlayAgain:   "Play Again",
		ExpressExit:        "Exit",
		ExpressDoubleClash: "DOUBLE CLASH!",
		ExpressTripleClash: "TRIPLE CLASH!",
		ExpressQuadClash:   "QUADRUPLE CLASH!",
		ExpressMegaClash:   "MEGA CLASH!",

		FishingTitle:     "Cozy Fishing",
		FishingSub:       "Winter Relax",
		FishingCast:      "CAST LINE",
		FishingReel:      "REEL IN!",
		FishingCatch:     "CATCH!",
		FishingMiss:      "Missed...",
		FishingWait:      "Waiting...",
		FishingScore:     "Fish Caught",
		FishingGameOver:  "Frozen Over!",
		FishingPlayAgain: "Fish Again",
		FishingExit:      "Warm Up",
	},
	"ua": {
		LobbyName:   "П'ЯТЬ3 Ігро-Space",
		Level:       "Рівень",
		XP:          "Досвід",
		DeployZone:  "Зона висадки",
		Shop:        "Крамниця",
		FriendsNav:  "Друзі",
		Customize:   "Кастомізація",
		Leaderboard: "Топгравців",
		Back:        "Назад",
		Settings:    "Налаштування",
		Menu:        "Меню",
		Close:       "Закрити",
		Cancel:      "Скасувати",
		Save:        "Зберегти",
		Later:       "Пізніше",
		Copy:        "Копіювати",
		LangCurrent: "UKR",

		Login:         "Увійти",
		Register:      "Реєстрація",
		Logout:        "Вийти",
		Nickname:      "Нікнейм",
		Password:      "Пароль",
		Tag:           "Теґ (напр. 0001)",
		RememberMe:    "Запам'ятати мене",
		Online:        "Онлайн",
		Away:          "Відійшов",
		Offline:       "Офлайн",
		CreateProfile: "Створити профіль",
		LoginTitle:    "Вхід",

		LangTitle:    "Мова",
		StatusTitle:  "Статус",
		AccountTitle: "Акаунт",
		LangNote:     "Ми збережемо це у твоєму профілі.",
		StatusNote:   "Порада: тисни на кружечок біля аватарки для швидкої зміни.",
		AccountNote:  "'Запам'ятати мене' дозволяє не вводити пароль щоразу.",

		AddFriendBtn:    "+ Додати друга",
		NoFriendsTitle:  "Поки що у тебе немає друзів",
		NoFriendsDesc:   "Додавай друзів за нікнеймом та теґом.",
		ChatAction:      "Чат",
		RemoveAction:    "Видалити",
		AddFriendHeader: "Додати друга",
		SendRequest:     "Надіслати",
		ChatTitle:       "Чат",

		CustomizeTitle:   "Кастомізація",
		NameColorTitle:   "Колір імені",
		BannerTitle:      "Банер лобі",
		PreviewLabel:     "ПЕРЕГЛЯД",
		ColorWhite:       "Звичайний білий",
		ColorGold:        "Сяючий золотавий",
		ColorRainbow:     "Райдужний",
		ColorHawkins:     "Вогники Гокінса",
		BannerDefault:    "Темний стандарт",
		BannerGold:       "Золоте сяйво",
		BannerCyber:      "Кіберпанк",
		BannerUpsideDown: "Виворіт",

		ExclusiveStyles: "Ексклюзив",
		Resources:       "Ресурси",
		ItemRainbowName: "Райдужний колір",
		ItemRainbowDesc: "Твій нікнейм переливатиметься всіма кольорами.",
		ItemGoldName:    "Золотий колір",
		ItemGoldDesc:    "Престижне золоте світіння.",
		ItemCyberBanner: "Кібербанер",
		ItemCyberDesc:   "Футуристичний фон для твого лобі.",
		ItemGoldBanner:  "Золотий банер",
		ItemGoldBannerD: "Покажи своє багатство.",
		ItemSack:        "Мішок Монет",
		ItemChest:       "Скриня Монет",

		LeaderboardTitle: "Таблиця лідерів",
		CurrentSeason:    "Поточний сезон",
		SeasonName:       "СЕЗОН 2: ЗИМОВІ ДИВА",
		EndsIn:           "Кінець через",
		Rank:             "Ранг",
		Trophies:         "Кубки",

		ChibikiTitle:    "Чібіки Рояль",
		ChibikiSub:      "Стратегія а-ля Clash Royale",
		BobikTitle:      "Бобік Шутер",
		BobikSub:        "Шутер від першої особи",
		PartyTitle:      "П'ЯТЬ3Ляп",
		PartySub:        "Паті-гейм (2-8 Гравців)",
		SlotixTitle:     "Слотікс",
		SlotixSub:       "Щасливий Автомат",
		UpsideDownTitle: "Виворіт",
		UpsideDownSub:   "Виживання у Дивних дивах",

		ExpressTitle:       "Новорічний Експрес",
		ExpressSub:         "Святкова головоломка",
		ExpressScore:       "Рахунок",
		ExpressGameOver:    "Кінець гри!",
		ExpressFinalScore:  "Твій фінальний рахунок: ",
		ExpressPlayAgain:   "Грати знову",
		ExpressExit:        "Вихід",
		ExpressDoubleClash: "ПОДВІЙНИЙ КЛЕШ!",
		ExpressTripleClash: "ПОТРІЙНИЙ КЛЕШ!",
		ExpressQuadClash:   "КВАДРО-КЛЕШ!",
		ExpressMegaClash:   "МЕГАКЛЕШ!",

		FishingTitle:     "Чумова рибалка",
		FishingSub:       "Зимовий релакс",
		FishingCast:      "ЗАКИНУТИ!",
		FishingReel:      "ПІДСІКТИ!",
		FishingCatch:     "СПІЙМАНО!",
		FishingMiss:      "Упущено...",
		FishingWait:      "Чекаємо...",
		FishingScore:     "Рибка на гачку",
		FishingGameOver:  "Замерзло!",
		FishingPlayAgain: "Знов за рибку гроші",
		FishingExit:      "Зігрітись",
	},
	"ru": {
		LobbyName:   "ПЯТЬ3 Игро-Space",
		Level:       "Уровень",
		XP:          "Опыт",
		DeployZone:  "Зона высадки",
		Shop:        "Магазин",
		FriendsNav:  "Друзья",
		Customize:   "Редактор",
		Leaderboard: "Лидерборд",
		Back:        "Назад",
		Settings:    "Настройки",
		Menu:        "Меню",
		Close:       "Закрыть",
		Cancel:      "Отмена",
		Save:        "Сохранить",
		Later:       "Позже",
		Copy:        "Копировать",
		LangCurrent: "RUS",

		Login:         "Войти",
		Register:      "Регистрация",
		Logout:        "Выйти",
		Nickname:      "Никнейм",
		Password:      "Пароль",
		Tag:           "Тег (напр. 0001)",
		RememberMe:    "Запомнить меня",
		Online:        "Онлайн",
		Away:          "Отошел",
		Offline:       "Оффлайн",
		CreateProfile: "Создать профиль",
		LoginTitle:    "Вход",

		LangTitle:    "Язык",
		StatusTitle:  "Статус",
		AccountTitle: "Аккаунт",
		LangNote:     "Мы сохраним это в твоем профиле.",
		StatusNote:   "Совет: нажми на круг у аватарки для быстрой смены.",
		AccountNote:  "'Запомнить меня' позволяет не вводить пароль каждый раз.",

		AddFriendBtn:    "+ Добавить друга",
		NoFriendsTitle:  "Пока нет друзей",
		NoFriendsDesc:   "Добавляй друзей по никнейму и тегу.",
		ChatAction:      "Чат",
		RemoveAction:    "Удалить",
		AddFriendHeader: "Добавить друга",
		SendRequest:     "Отправить",
		ChatTitle:       "Чат",

		CustomizeTitle:   "Редактор",
		NameColorTitle:   "Цвет имени",
		BannerTitle:      "Баннер лобби",
		PreviewLabel:     "ПРЕДПРОСМОТР",
		ColorWhite:       "Обычный белый",
		ColorGold:        "Золотистый блеск",
		ColorRainbow:     "Радужное осуждение",
		ColorHawkins:     "Огоньки Хокинса",
		BannerDefault:    "Тёмный стандарт",
		BannerGold:       "Золотое сияние",
		BannerCyber:      "Киберпанк",
		BannerUpsideDown: "Изнанка",

		ExclusiveStyles: "Эксклюзив",
		Resources:       "Ресурсы",
		ItemRainbowName: "Радужный цвет",
		ItemRainbowDesc: "Твой никнейм будет переливаться всеми цветами.",
		ItemGoldName:    "Золотой цвет",
		ItemGoldDesc:    "Престижное золотое свечение.",
		ItemCyberBanner: "Кибербаннер",
		ItemCyberDesc:   "Футуристичный фон для твоего лобби.",
		ItemGoldBanner:  "Богатый баннер",
		ItemGoldBannerD: "Покажи своё богатство.",
		ItemSack:        "Мешок Монет",
		ItemChest:       "Сундук Монет",

		LeaderboardTitle: "Таблица Лидеров",
		CurrentSeason:    "Текущий Сезон",
		SeasonName:       "СЕЗОН 2: ЗИМНИЕ ДЕЛА",
		EndsIn:           "Конец через",
		Rank:             "Ранг",
		Trophies:         "Кубки",

		ChibikiTitle:    "Чибики Рояль",
		ChibikiSub:      "Стратегия а ля Clash Royale",
		BobikTitle:      "Бобик Шутер",
		BobikSub:        "Шутер от первого лица",
		PartyTitle:      "Пять3Ёбка",
		PartySub:        "Пати-гейм (2-8 Игроков)",
		SlotixTitle:     "Слотикс",
		SlotixSub:       "Счастливый Автомат",
		UpsideDownTitle: "Изнанка",
		UpsideDownSub:   "Выживание в Stranger Things",

		ExpressTitle:       "Новогодний Экспресс",
		ExpressSub:         "Праздничная головоломка",
		ExpressScore:       "Счет",
		ExpressGameOver:    "Игра окончена!",
		ExpressFinalScore:  "Твой финальный счет: ",
		ExpressPlayAgain:   "Играть снова",
		ExpressExit:        "Выход",
		ExpressDoubleClash: "ДВОЙНОЙ КЛЕШ!",
		ExpressTripleClash: "ТРОЙНОЙ КЛЕШ!",
		ExpressQuadClash:   "КВАДРО-КЛЕШ!",
		ExpressMegaClash:   "МЕГАКЛЕШ!",

		FishingTitle:     "Чумовая Рыбалка",
		FishingSub:       "Зимний Релакс",
		FishingCast:      "ЗАБРОСИТЬ",
		FishingReel:      "ТЯНИ!",
		FishingCatch:     "ЕСТЬ!",
		FishingMiss:      "Сорвалась...",
		FishingWait:      "Ждем...",
		FishingScore:     "Рыбы поймано",
		FishingGameOver:  "Замерзло!",
		FishingPlayAgain: "Рыбачить еще",
		FishingExit:      "Греться",
	},
}
