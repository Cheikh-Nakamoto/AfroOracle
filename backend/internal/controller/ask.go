package controller

import (
	"fmt"
	"net/http"
	"time"

	"afrioracle/internal/view"
)

func (c *Controllers) HandleAsk(w http.ResponseWriter, r *http.Request) {
	publications := c.Store.Publications()
	if len(publications) == 0 {
		view.WriteJSON(w, http.StatusOK, map[string]string{
			"answer": "Aucune donnée publiée pour le moment. Je n'estime jamais un prix moi-même.",
		})
		return
	}
	last := publications[len(publications)-1]
	answer := fmt.Sprintf("Prix du mil à Kaolack : %.0f FCFA/kg, publié le %s, %d sources, confiance %d. Tx : %s",
		last.Median, time.Unix(last.Timestamp, 0).Format("15:04"), c.Store.AcceptedCount(), last.Confidence, last.Tx)
	view.WriteJSON(w, http.StatusOK, map[string]string{"answer": answer})
}
