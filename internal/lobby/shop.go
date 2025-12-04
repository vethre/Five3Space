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
	Currency string `json:"currency"` // For logging/analytics
}

func NewBuyHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Auth Check
		c, err := r.Cookie("user_id")
		if err != nil || c.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := c.Value

		// 2. Parse Request
		var req BuyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// 3. Process Item
		// In a real app, you'd fetch item stats from DB. Here we hardcode logic.
		var successMsg string
		var newBalance int

		switch req.ItemID {
		// --- REAL MONEY PACKS (Mock Payment) ---
		case "coins_1000", "pack_support":
			// Simulate successful payment -> Give 1000 Coins
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

		// --- COSMETICS (Coin Purchase) ---
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

		default:
			http.Error(w, "Unknown Item", http.StatusBadRequest)
			return
		}

		// 4. Return New State
		// Fetch fresh balance if we didn't get it from processCoinPurchase
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

// Helper for coin items
func processCoinPurchase(store *data.Store, userID, itemID string, cost int) (int, error) {
	// 1. Check Balance
	user, ok := store.GetUser(userID)
	if !ok {
		return 0, fmt.Errorf("User not found")
	}
	if user.Coins < cost {
		return 0, fmt.Errorf("Not enough coins!")
	}

	// 2. Check if already owned
	if store.HasItem(userID, itemID) {
		return 0, fmt.Errorf("You already own this item")
	}

	// 3. Transaction: Deduct Coins & Add Item
	if err := store.DeductCoinsAndAddItem(userID, itemID, cost); err != nil {
		log.Println("Purchase error:", err)
		return 0, fmt.Errorf("Transaction failed")
	}

	return user.Coins - cost, nil
}
