package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaeman-dev/kaeman-api/shared/utils"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("error: ")

	reader := bufio.NewReader(os.Stdin)
	prompt := func(label string) string {
		fmt.Print(label)

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		return strings.TrimSpace(line)
	}

	accessToken := prompt("access token: ")

	serverID := prompt("server id: ")
	if accessToken == "" || serverID == "" {
		log.Fatal("access token and server id are required")
	}

	var claims struct {
		jwt.RegisteredClaims
		Profiles struct {
			MC string `json:"mc"`
		} `json:"profiles"`
	}
	if _, _, err := jwt.NewParser().ParseUnverified(accessToken, &claims); err != nil {
		log.Fatal(err)
	}

	if claims.Profiles.MC == "" {
		log.Fatal("access token carries no minecraft profile")
	}

	profile, err := utils.NormalizeUUID(claims.Profiles.MC)
	if err != nil {
		log.Fatal(err)
	}

	body, err := json.Marshal(map[string]string{
		"accessToken":     accessToken,
		"selectedProfile": profile,
		"serverId":        serverID,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Post("https://sessionserver.mojang.com/session/minecraft/join", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	response, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	closeErr := resp.Body.Close()

	if err != nil {
		log.Fatal(err)
	}

	if closeErr != nil {
		log.Fatal(closeErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Fatalf("POST session/minecraft/join: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(response)))
	}

	fmt.Println("joined:", profile)
}
