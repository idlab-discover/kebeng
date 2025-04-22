package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterName(c *gin.Context) {
	data := map[string]string{
		"snap_name": "sinus123",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post("/dev/api/register-name/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("HTTP error:", err)
		return
	}
	defer resp.Body.Close()
}
