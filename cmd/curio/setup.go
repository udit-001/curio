package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ---- Terminal helpers ----

var (
	stdinReader = bufio.NewReader(os.Stdin)
)

func termBold() string {
	if termSupportsColor() {
		return "\033[1m"
	}
	return ""
}

func termDim() string {
	if termSupportsColor() {
		return "\033[2m"
	}
	return ""
}

func termBlue() string {
	if termSupportsColor() {
		return "\033[34m"
	}
	return ""
}

func termGreen() string {
	if termSupportsColor() {
		return "\033[32m"
	}
	return ""
}

func termYellow() string {
	if termSupportsColor() {
		return "\033[33m"
	}
	return ""
}

func termReset() string {
	if termSupportsColor() {
		return "\033[0m"
	}
	return ""
}

func termSupportsColor() bool {
	if !isTerminal() {
		return false
	}
	return true
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func clearScreen() {
	if !isTerminal() {
		return
	}
	fmt.Print("\033[2J\033[3J\033[H")
}

func openBrowser(url string) {
	fmt.Printf("  %s↗ opening%s %s\n", termGreen(), termReset(), url)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Printf("  %s⚠ couldn't open browser — visit it manually: %s%s\n", termYellow(), url, termReset())
		return
	}
	_ = cmd.Start()
}

func say(msg string) {
	fmt.Printf("  %s\n", msg)
}

func step(msg string) {
	fmt.Printf("  %s•%s %s\n", termBlue(), termReset(), msg)
}

func note(msg string) {
	fmt.Printf("  %s%s%s\n", termDim(), msg, termReset())
}

func warn(msg string) {
	fmt.Printf("  %s⚠ %s%s\n", termYellow(), msg, termReset())
}

func success(msg string) {
	fmt.Printf("  %s✓ %s%s\n", termGreen(), msg, termReset())
}

func pause(msg string) {
	if msg == "" {
		msg = "Press Enter to continue"
	}
	fmt.Printf("  %s%s%s ", termDim(), msg, termReset())
	_, _ = stdinReader.ReadString('\n')
}

func confirm(question string) bool {
	fmt.Printf("  %s? %s [y/N] %s", termYellow(), question, termReset())
	reply, _ := stdinReader.ReadString('\n')
	reply = strings.TrimSpace(reply)
	return strings.HasPrefix(strings.ToLower(reply), "y")
}

func ask(prompt string) string {
	fmt.Printf("  %s%s%s ", termBold(), prompt, termReset())
	input, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(input)
}

func askWithDefault(prompt, key string) string {
	current := configGet(key)
	if current != "" {
		fmt.Printf("  %s%s%s %s[Enter keeps current]%s ", termBold(), prompt, termReset(), termDim(), termReset())
	} else {
		fmt.Printf("  %s%s%s ", termBold(), prompt, termReset())
	}
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" && current != "" {
		return current
	}
	return input
}

func askSecret(prompt string) string {
	fmt.Printf("  %s%s%s %s(paste + Enter, hidden)%s ", termBold(), prompt, termReset(), termDim(), termReset())
	oldState, _ := makeStdinHidden()
	input, _ := stdinReader.ReadString('\n')
	restoreStdin(oldState)
	input = strings.TrimSpace(input)
	fmt.Printf("\r  %s✓ received%s\n", termGreen(), termReset())
	return input
}

func askSecretWithDefault(prompt, key string) string {
	current := configGet(key)
	if current != "" {
		fmt.Printf("  %s%s%s %s[Enter keeps current, paste to replace — hidden]%s ", termBold(), prompt, termReset(), termDim(), termReset())
	} else {
		fmt.Printf("  %s%s%s %s(paste + Enter, hidden)%s ", termBold(), prompt, termReset(), termDim(), termReset())
	}
	oldState, _ := makeStdinHidden()
	input, _ := stdinReader.ReadString('\n')
	restoreStdin(oldState)
	input = strings.TrimSpace(input)
	fmt.Printf("\r  %s✓ received%s\n", termGreen(), termReset())
	if input == "" && current != "" {
		return current
	}
	return input
}

func makeStdinHidden() (interface{}, error) {
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if runtime.GOOS == "windows" {
		return nil, nil
	}
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(getTermiosIoctl()), uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	newState := oldState
	newState.Lflag &^= syscall.ECHO
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(setTermiosIoctl()), uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	return oldState, nil
}

func restoreStdin(state interface{}) {
	if state == nil {
		return
	}
	fd := int(os.Stdin.Fd())
	if oldState, ok := state.(syscall.Termios); ok {
		_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(setTermiosIoctl()), uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
	}
}

// ---- Stage table ----

type stageConfig struct {
	name            string
	key             string // config key for "configured" check
	desc            string // short description for the header
	minutes         int
	url             string   // signup URL
	instructions    []string // step messages after URL
	prompt          string   // prompt for the secret key
	testFunc        func(key string) error
	secondURL       string   // optional second URL (Europeana)
	secondSteps     []string // steps after second URL
	pauseAfterFirst bool     // pause between URL blocks
	userKey         string   // optional non-secret field (Wikimedia)
	userPrompt      string
	secretKey       string // if set, secret goes here instead of key
	skipNote        string // default: "source will be unavailable"
	successNote     string // overrides default success message
}

var wizardStages = []stageConfig{
	{
		name:         "Wikimedia",
		key:          "wikimedia.bot_user",
		desc:         "bot password (optional)",
		minutes:      3,
		url:          "https://commons.wikimedia.org/wiki/Special:BotPasswords",
		instructions: []string{"Log in to Wikimedia Commons if prompted.", "Enter a bot name (e.g. 'curio') and create it.", "Check these permissions: High-volume (bot), Existing, Edit existing.", "You'll get a password (shown once). Copy it."},
		userKey:      "wikimedia.bot_user",
		userPrompt:   "Bot login username (e.g. YourName@curio):",
		secretKey:    "wikimedia.bot_pass",
		prompt:       "Bot password:",
		skipNote:     "using keyless with maxlag",
		successNote:  "Credentials saved — higher Wikimedia limits active.",
	},
	{
		name:         "Smithsonian",
		key:          "smithsonian.api_key",
		desc:         "api.data.gov key",
		minutes:      2,
		url:          "https://api.data.gov/signup/",
		instructions: []string{"Fill in the form (email + name). The key is shown immediately.", "Copy the API key."},
		prompt:       "Your api.data.gov key:",
		testFunc:     testSmithsonian,
		successNote:  "Key verified — Smithsonian unlocked (1000 req/hour).",
	},
	{
		name:            "Europeana",
		key:             "europeana.api_key",
		desc:            "API key",
		minutes:         2,
		url:             "https://www.europeana.eu/account/login",
		instructions:    []string{"Create an account or log in if you already have one."},
		pauseAfterFirst: true,
		secondURL:       "https://www.europeana.eu/en/account/api-keys",
		secondSteps:     []string{"Create a personal API key and copy it."},
		prompt:          "Your Europeana API key:",
		testFunc:        testEuropeana,
	},
	{
		name:         "Pexels",
		key:          "pexels.api_key",
		desc:         "API key",
		minutes:      2,
		url:          "https://www.pexels.com/api/new/",
		instructions: []string{"Fill in your details. The API key is shown immediately."},
		prompt:       "Your Pexels API key:",
		testFunc:     testPexels,
	},
	{
		name:         "Pixabay",
		key:          "pixabay.api_key",
		desc:         "API key",
		minutes:      2,
		url:          "https://pixabay.com/accounts/register/",
		instructions: []string{"Create an account, then go to your Account Settings → API.", "Copy your API key."},
		prompt:       "Your Pixabay API key:",
		testFunc:     testPixabay,
	},
	{
		name:         "Unsplash",
		key:          "unsplash.access_key",
		desc:         "API key",
		minutes:      2,
		url:          "https://unsplash.com/oauth/applications",
		instructions: []string{"Log in or create an account.", "Click 'New Application' and accept the terms.", "Fill in a name/description, then copy your Access Key."},
		prompt:       "Your Unsplash Access Key:",
		testFunc:     testUnsplash,
	},
	{
		name:         "BHL",
		key:          "bhl.api_key",
		desc:         "API key",
		minutes:      2,
		url:          "https://www.biodiversitylibrary.org/getapikey.aspx",
		instructions: []string{"Fill in the form. The API key is shown immediately."},
		prompt:       "Your BHL API key:",
		testFunc:     testBHL,
	},
}

// runStage executes a single setup stage using the data-driven template.
// Returns the updated minutesElapsed.
func runStage(s stageConfig, stageIdx, totalStages, minutesElapsed, totalMinutes int) int {
	minutesElapsed += s.minutes
	clearScreen()
	remaining := totalMinutes - minutesElapsed
	fmt.Printf("\n%s%s▸ Stage %d/%d · %s — %s%s  %s(~%d min left)%s\n",
		termBold(), termBlue(), stageIdx, totalStages, s.name, s.desc, termReset(), termDim(), remaining, termReset())

	if configGet(s.key) != "" {
		success(fmt.Sprintf("%s already configured — skipping.", s.name))
		pause("")
		return minutesElapsed
	}

	if !confirm(fmt.Sprintf("Set up %s authentication?", s.name)) {
		skipMsg := s.skipNote
		if skipMsg == "" {
			skipMsg = "source will be unavailable"
		}
		note(fmt.Sprintf("Skipping %s — %s.", s.name, skipMsg))
		pause("")
		return minutesElapsed
	}

	openBrowser(s.url)
	for _, ins := range s.instructions {
		step(ins)
	}

	if s.secondURL != "" {
		if s.pauseAfterFirst {
			pause("Done?")
		}
		openBrowser(s.secondURL)
		for _, ins := range s.secondSteps {
			step(ins)
		}
	}

	fmt.Println()

	// Optional: non-secret username field (Wikimedia)
	var userVal string
	if s.userPrompt != "" {
		userVal = askWithDefault(s.userPrompt, s.userKey)
	}

	// Secret input
	secretKey := s.key
	if s.secretKey != "" {
		secretKey = s.secretKey
	}
	apiKey := askSecretWithDefault(s.prompt, secretKey)

	// Validate
	if s.userPrompt != "" && userVal == "" {
		warn(fmt.Sprintf("Missing username — skipping %s.", s.name))
		pause("")
		return minutesElapsed
	}
	if apiKey == "" {
		warn(fmt.Sprintf("No key provided — skipping %s.", s.name))
		pause("")
		return minutesElapsed
	}

	// Write username if applicable
	if s.userPrompt != "" {
		configSet(s.userKey, userVal)
	}

	// Test the key
	if s.testFunc != nil {
		note("Testing key...")
		if err := s.testFunc(apiKey); err != nil {
			warn(fmt.Sprintf("Key test failed — saving anyway. (%v)", err))
			configSet(secretKey, apiKey)
		} else {
			configSet(secretKey, apiKey)
			msg := s.successNote
			if msg == "" {
				msg = fmt.Sprintf("Key verified — %s unlocked.", s.name)
			}
			success(msg)
		}
	} else {
		configSet(secretKey, apiKey)
		msg := s.successNote
		if msg == "" {
			msg = fmt.Sprintf("Credentials saved — %s configured.", s.name)
		}
		success(msg)
	}

	pause("")
	return minutesElapsed
}

// ---- Key test functions ----

func testSmithsonian(key string) error {
	resp, err := http.Get("https://api.si.edu/openaccess/api/v1.0/stats?api_key=" + key)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func testEuropeana(key string) error {
	var result struct {
		Success bool `json:"success"`
	}
	u := "https://api.europeana.eu/record/v2/search.json?query=test&wskey=" + key + "&rows=0"
	if err := httpGetJSON(u, nil, &result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("success=false")
	}
	return nil
}

func testPexels(key string) error {
	resp, err := httpGet("https://api.pexels.com/v1/search?query=test&per_page=1", map[string]string{"Authorization": key})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func testPixabay(key string) error {
	var result struct {
		TotalHits int `json:"totalHits"`
	}
	u := "https://pixabay.com/api/?key=" + key + "&q=test&per_page=1"
	if err := httpGetJSON(u, nil, &result); err != nil {
		return err
	}
	return nil
}

func testUnsplash(key string) error {
	resp, err := httpGet("https://api.unsplash.com/search/photos?query=test&per_page=1", map[string]string{"Authorization": "Client-ID " + key})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func testBHL(key string) error {
	resp, err := http.Get("https://www.biodiversitylibrary.org/api3?action=GetStatus&format=json&apikey=" + key)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ---- Setup wizard ----

func runSetup() {
	// Check what's already configured
	type sourceStatus struct {
		name       string
		key        string
		configured bool
	}
	statuses := []sourceStatus{
		{"Openverse", "openverse.client_id", false},
		{"Wikimedia", "wikimedia.bot_user", false},
		{"Smithsonian", "smithsonian.api_key", false},
		{"Europeana", "europeana.api_key", false},
		{"Pexels", "pexels.api_key", false},
		{"Pixabay", "pixabay.api_key", false},
		{"Unsplash", "unsplash.access_key", false},
		{"BHL", "bhl.api_key", false},
	}
	allConfigured := true
	anyConfigured := false
	for i := range statuses {
		statuses[i].configured = configGet(statuses[i].key) != ""
		if !statuses[i].configured {
			allConfigured = false
		} else {
			anyConfigured = true
		}
	}

	// If everything is configured, show status and ask before proceeding
	if allConfigured {
		clearScreen()
		fmt.Printf("\n%s%s  curio — setup%s\n\n", termBold(), termBlue(), termReset())
		note("All sources are configured:")
		fmt.Println()
		for _, s := range statuses {
			fmt.Printf("  %s✓%s %s\n", termGreen(), termReset(), s.name)
		}
		fmt.Println()
		if !confirm("Re-run setup to update any keys?") {
			note("Nothing to do. Keys are at: " + ConfigPath)
			fmt.Println()
			return
		}
	}

	// If some are configured, show status before the wizard
	if anyConfigured && !allConfigured {
		clearScreen()
		fmt.Printf("\n%s%s  curio — setup%s\n\n", termBold(), termBlue(), termReset())
		for _, s := range statuses {
			if s.configured {
				fmt.Printf("  %s✓%s %s — configured\n", termGreen(), termReset(), s.name)
			} else {
				fmt.Printf("  %s✗%s %s — not configured\n", termYellow(), termReset(), s.name)
			}
		}
		fmt.Println()
		note("Only unconfigured sources will be walked through.")
		fmt.Println()
	}

	totalStages := 1 + len(wizardStages) // Openverse + table stages
	totalMinutes := 4                    // Openverse
	for _, s := range wizardStages {
		totalMinutes += s.minutes
	}
	stageIdx := 0
	minutesElapsed := 0

	if !allConfigured {
		clearScreen()
		fmt.Printf("\n%s%s  curio — setup%s\n", termBold(), termBlue(), termReset())
		fmt.Printf("%s  %d stages · about %d minutes%s\n\n", termDim(), totalStages, totalMinutes, termReset())
		fmt.Printf("%s  This wizard configures API keys for higher rate limits and access to\n", termDim())
		fmt.Printf("  key-required sources. All keys are optional — keyless sources work\n")
		fmt.Printf("  without any setup. Stop any time with Ctrl-C.%s\n", termReset())
		pause("Ready to start?")
	}

	// ── Stage 1: Openverse (custom — programmatic app registration) ──────
	stageIdx++
	minutesElapsed += 4
	clearScreen()
	remaining := totalMinutes - minutesElapsed
	fmt.Printf("\n%s%s▸ Stage %d/%d · Openverse — register app%s  %s(~%d min left)%s\n",
		termBold(), termBlue(), stageIdx, totalStages, termReset(), termDim(), remaining, termReset())
	if configGet("openverse.client_id") != "" {
		success("Openverse already configured — skipping.")
		pause("")
	} else if !confirm("Set up Openverse authentication?") {
		note("Skipping Openverse — staying keyless (200/day).")
		pause("")
	} else {
		step("This registers an app programmatically. You'll need an email address.")
		fmt.Println()
		email := ask("Your email (for Openverse verification):")

		if email == "" {
			warn("No email provided — skipping Openverse.")
		} else {
			note("Registering app 'curio' with Openverse...")
			regResp, err := http.PostForm("https://api.openverse.org/v1/auth_tokens/register/", url.Values{
				"name":        {fmt.Sprintf("curio-%d", time.Now().UnixNano())},
				"description": {"curio skill"},
				"email":       {email},
			})
			if err != nil {
				warn(fmt.Sprintf("Registration failed: %v", err))
			} else {
				var reg struct {
					ClientID     string `json:"client_id"`
					ClientSecret string `json:"client_secret"`
				}
				json.NewDecoder(regResp.Body).Decode(&reg)
				regResp.Body.Close()

				if reg.ClientID == "" || reg.ClientSecret == "" {
					warn("Registration didn't return credentials.")
				} else {
					note(fmt.Sprintf("Registered! A verification link was sent to %s.", email))
					fmt.Println()
					step("Open your email and click the Openverse verification link.")
					step("Come back here after clicking it.")
					pause("Clicked the verification link?")

					note("Testing credentials...")
					tokenResp, err := http.PostForm("https://api.openverse.org/v1/auth_tokens/token/", url.Values{
						"client_id":     {reg.ClientID},
						"client_secret": {reg.ClientSecret},
						"grant_type":    {"client_credentials"},
					})
					if err != nil {
						warn(fmt.Sprintf("Token exchange failed: %v", err))
						configSet("openverse.client_id", reg.ClientID)
						configSet("openverse.client_secret", reg.ClientSecret)
					} else {
						var tok struct {
							AccessToken string `json:"access_token"`
						}
						json.NewDecoder(tokenResp.Body).Decode(&tok)
						tokenResp.Body.Close()

						if tok.AccessToken != "" {
							configSet("openverse.client_id", reg.ClientID)
							configSet("openverse.client_secret", reg.ClientSecret)
							success("Credentials verified — 10,000 requests/day unlocked.")
						} else {
							warn("Token exchange failed — credentials saved anyway.")
							configSet("openverse.client_id", reg.ClientID)
							configSet("openverse.client_secret", reg.ClientSecret)
						}
					}
				}
			}
		}
		pause("")
	}

	// ── Stages 2-8: data-driven table ───────────────────────────────────
	for _, s := range wizardStages {
		stageIdx++
		minutesElapsed = runStage(s, stageIdx, totalStages, minutesElapsed, totalMinutes)
	}

	// ── Summary ─────────────────────────────────────────────────────────
	clearScreen()
	fmt.Printf("\n%s%s  ✓ Setup complete%s\n", termBold(), termGreen(), termReset())
	fmt.Println()
	note(fmt.Sprintf("Config: %s", ConfigPath))
	fmt.Println()

	sourceKeys := []struct{ name, key string }{
		{"Openverse", "openverse.client_id"},
		{"Wikimedia", "wikimedia.bot_user"},
		{"Smithsonian", "smithsonian.api_key"},
		{"Europeana", "europeana.api_key"},
		{"Pexels", "pexels.api_key"},
		{"Pixabay", "pixabay.api_key"},
		{"Unsplash", "unsplash.access_key"},
		{"BHL", "bhl.api_key"},
	}
	for _, s := range sourceKeys {
		if configGet(s.key) != "" {
			fmt.Printf("  %s✓%s %s\n", termGreen(), termReset(), s.name)
		} else {
			fmt.Printf("  %s✗%s %s\n", termYellow(), termReset(), s.name)
		}
	}
	fmt.Println()
	say("You can re-run this wizard any time: curio setup")
	say(fmt.Sprintf("To go back to keyless mode: rm %s", ConfigPath))
	fmt.Println()
}

// ---- Platform-specific helpers for hidden input ----

func getTermiosIoctl() uintptr {
	if runtime.GOOS == "darwin" {
		return 0x40487413 // TIOCGETA on macOS
	}
	return 0x5401 // TCGETS on Linux
}

func setTermiosIoctl() uintptr {
	if runtime.GOOS == "darwin" {
		return 0x80487414 // TIOCSETA on macOS
	}
	return 0x5402 // TCSETS on Linux
}
