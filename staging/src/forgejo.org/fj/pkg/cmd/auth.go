package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"

	"forgejo.org/fj/internal/keys"
	"github.com/spf13/cobra"
)

// Builtin OAuth client IDs for known Forgejo instances.
// For instances without a builtin (like git.rezus.cloud), the user can
// provide a token via `fj auth add-key <token>`.
var builtinClientIDs = map[string]string{
	"codeberg.org":            "19ac3dd0-e101-445d-aa60-d8ea3876bc5d",
	"code.forgejo.org":        "ab67d8a2-72bd-42e8-ae05-937eaba31e24",
	"v16.next.forgejo.org":    "0b561d01-fd05-4321-9d46-9cb8c776fc80",
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthListCmd())
	cmd.AddCommand(newAuthAddKeyCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to a Forgejo instance (browser OAuth or token prompt)",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			if host == "" {
				return fmt.Errorf("--host is required")
			}

			ki, err := keys.Load()
			if err != nil {
				return err
			}

			// Try OAuth for known instances
			if clientID, ok := builtinClientIDs[host]; ok {
				return oauthLogin(ki, host, clientID)
			}

			// Fallback: prompt for an application token
			fmt.Printf("No OAuth client configured for %s.\n", host)
			fmt.Print("Enter an application token (from Settings > Applications): ")
			var token string
			fmt.Scanln(&token)
			if token == "" {
				return fmt.Errorf("no token provided")
			}
			ki.SetLogin(host, &keys.LoginInfo{Type: "Application", Token: token})
			if err := ki.Save(); err != nil {
				return err
			}
			fmt.Printf("Logged in to %s\n", host)
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <HOST>",
		Short: "Log out of a Forgejo instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host := args[0]
			ki, err := keys.Load()
			if err != nil {
				return err
			}
			if !ki.RemoveLogin(host) {
				return fmt.Errorf("not logged in to %s", host)
			}
			if err := ki.Save(); err != nil {
				return err
			}
			fmt.Printf("Logged out of %s\n", host)
			return nil
		},
	}
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all authenticated hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ki, err := keys.Load()
			if err != nil {
				return err
			}
			hosts := ki.ListHosts()
			if len(hosts) == 0 {
				fmt.Println("No authenticated hosts. Run: fj auth login")
				return nil
			}
			for _, h := range hosts {
				login := ki.GetLogin(h)
				fmt.Printf("  %s (type: %s)\n", h, login.Type)
			}
			return nil
		},
	}
}

func newAuthAddKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-key [TOKEN]",
		Short: "Add an application token for the current --host",
		RunE: func(cmd *cobra.Command, args []string) error {
			host, _ := cmd.Flags().GetString("host")
			if host == "" {
				return fmt.Errorf("--host is required")
			}
			var token string
			if len(args) > 0 {
				token = args[0]
			} else {
				// Read from stdin
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				token = string(data)
				token = token[:len(token)-1] // strip newline
			}
			if token == "" {
				return fmt.Errorf("no token provided")
			}
			ki, err := keys.Load()
			if err != nil {
				return err
			}
			ki.SetLogin(host, &keys.LoginInfo{Type: "Application", Token: token})
			if err := ki.Save(); err != nil {
				return err
			}
			fmt.Printf("Added token for %s\n", host)
			return nil
		},
	}
}

// oauthLogin performs the browser-based OAuth2 flow:
// 1. Start a local HTTP server on :26218 to receive the callback
// 2. Open the browser to the forgejo authorize URL
// 3. Exchange the auth code for an access token
// 4. Store the token in keys.json
func oauthLogin(ki *keys.KeyInfo, host, clientID string) error {
	redirectURI := "http://127.0.0.1:26218/"
	state := fmt.Sprintf("%d", os.Getpid())

	// Build authorize URL
	authURL := fmt.Sprintf("https://%s/login/oauth/authorize?%s", host,
		url.Values{
			"client_id":     {clientID},
			"redirect_uri":  {redirectURI},
			"response_type": {"code"},
			"state":         {state},
		}.Encode())

	// Channel to receive the auth code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local server
	server := &http.Server{Addr: ":26218"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		fmt.Fprintf(w, "Authentication successful. You can close this tab.")
		codeCh <- code
	})

	go server.ListenAndServe()
	defer server.Close()

	// Open browser
	fmt.Printf("Opening browser to log in to %s...\n", host)
	openBrowser(authURL)

	// Wait for callback
	select {
	case code := <-codeCh:
		// Exchange code for token
		tokenResp, err := exchangeCode(host, clientID, code, redirectURI)
		if err != nil {
			return fmt.Errorf("exchanging code: %w", err)
		}
		ki.SetLogin(host, &keys.LoginInfo{
			Type:         "OAuth",
			Token:        tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			ExpiresAt:    0, // TODO: compute from expires_in
		})
		if err := ki.Save(); err != nil {
			return err
		}
		fmt.Printf("Logged in to %s\n", host)
		return nil
	case err := <-errCh:
		return err
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func exchangeCode(host, clientID, code, redirectURI string) (*tokenResponse, error) {
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}.Encode()
	resp, err := http.Post(fmt.Sprintf("https://%s/login/oauth/access_token", host), "application/x-www-form-urlencoded", stringReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s: %s", resp.Status, respBody)
	}
	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func stringReader(s string) io.Reader {
	return &stringReaderImpl{s: s}
}

type stringReaderImpl struct {
	s   string
	pos int
}

func (r *stringReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

// helper: make context available for future use
var _ = context.Background
