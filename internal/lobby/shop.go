package lobby

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"main/internal/data"
)

type BuyRequest struct {
	ItemID   string `json:"item_id"`
	Currency string `json:"currency"`
}

func NewBuyHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("user_id")
		if err != nil || c.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := c.Value

		var req BuyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var successMsg string
		var newBalance int

		switch req.ItemID {
		case "coins_1000", "pack_support":
			if err := store.AdjustCoins(userID, 1000); err != nil {
				http.Error(w, "DB Error", http.StatusInternalServerError)
				return
			}
			successMsg = "Payment Successful! +1000 Coins"

		case "coins_5000", "pack_founder":
			if err := store.AdjustCoins(userID, 5000); err != nil {
				http.Error(w, "DB Error", http.StatusInternalServerError)
				return
			}
			successMsg = "Payment Successful! +5000 Coins"

		// --- COSMETICS ---
		case "frame_neon":
			newBalance, err = processCoinPurchase(store, userID, "frame_neon", 2500)
			if err != nil {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			successMsg = "Neon Frame Purchased!"

		case "banner_gold":
			newBalance, err = processCoinPurchase(store, userID, "banner_gold", 5000)
			if err != nil {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			successMsg = "Gold Banner Purchased!"

		case "name_rainbow":
			newBalance, err = processCoinPurchase(store, userID, "name_rainbow", 8000)
			if err != nil {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			successMsg = "Rainbow Name Purchased!"

		case "name_gold":
			newBalance, err = processCoinPurchase(store, userID, "name_gold", 4000)
			if err != nil {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			successMsg = "Gold Name Purchased!"

		case "banner_cyber":
			newBalance, err = processCoinPurchase(store, userID, "banner_cyber", 3500)
			if err != nil {
				http.Error(w, err.Error(), http.StatusPaymentRequired)
				return
			}
			successMsg = "Cyber Banner Purchased!"

		default:
			http.Error(w, "Unknown Item", http.StatusBadRequest)
			return
		}

		if newBalance == 0 {
			u, _ := store.GetUser(userID)
			newBalance = u.Coins
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": successMsg,
			"coins":   newBalance,
		})
	}
}

func processCoinPurchase(store *data.Store, userID, itemID string, cost int) (int, error) {
	user, ok := store.GetUser(userID)
	if !ok {
		return 0, fmt.Errorf("User not found")
	}
	if user.Coins < cost {
		return 0, fmt.Errorf("Not enough coins!")
	}
	if store.HasItem(userID, itemID) {
		return 0, fmt.Errorf("You already own this item")
	}

	if err := store.DeductCoinsAndAddItem(userID, itemID, cost); err != nil {
		log.Println("Purchase error:", err)
		return 0, fmt.Errorf("Transaction failed")
	}
	return user.Coins - cost, nil
}
