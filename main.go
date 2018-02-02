package main

import (
	"fmt"
	"net/http"
	"log"
	"net/url"
	"encoding/json"
	"os"
)

var (
	clientId            = os.Getenv("CLIENT_ID")
	clientSecret        = os.Getenv("CLIENT_SECRET")
	callbackUrl         = os.Getenv("CALLBACK_URL")
	allowedHostedDomain = os.Getenv("ALLOWED_DOMAIN")
)

const oauthUrl = "https://accounts.google.com/o/oauth2/auth?redirect_uri=%s&response_type=code&client_id=%s&scope=openid+email+profile&approval_prompt=force&access_type=offline"
const tokenUrl = "https://www.googleapis.com/oauth2/v3/token"
const userInfoUrl = "https://www.googleapis.com/oauth2/v1/userinfo"
const idpIssuerUrl = "https://accounts.google.com"

func main() {
	log.Println("Starting")

	httpPort := "80"

	m := http.NewServeMux()

	m.Handle("/", GetOauth())
	m.Handle("/callback", GetOauthCallback())
	log.Println("Listening on port", httpPort)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", httpPort), m))
}

func GetOauth() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf(oauthUrl, callbackUrl, clientId), http.StatusFound)
	})
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

type UserEmail struct {
	Email string `json:"email"`
}

type HostedDomain struct {
	HostedDomain string `json:"hd"`
}

func GetOauthCallback() http.Handler {

	runHelp := `
# Run the following command to configure a kubernetes user for use with 'kubectl'
# Go to to your kube config and update user under context part to:
# '''
# user: %s
# '''
	`

	kubectlCMDTemplate := `
kk{environment} config set-credentials %s \
--auth-provider=oidc \
--auth-provider-arg=client-id=%s \
--auth-provider-arg=client-secret=%s \
--auth-provider-arg=id-token=%s \
--auth-provider-arg=idp-issuer-url=%s \
--auth-provider-arg=refresh-token=%s
	`
	outputTemplate := runHelp + kubectlCMDTemplate
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")

		tokenResponse, err := GetTokens(code)

		if err != nil {
			log.Printf("Error getting tokens: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return

		}

		email, err := GetUserEmail(tokenResponse.AccessToken)

		if err != nil {
			log.Printf("Error getting user email: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hostedDomain, err := getHostedDomain(tokenResponse.AccessToken)
		if hostedDomain != allowedHostedDomain {
			log.Printf("Error hosted domain does not match (was %s instead of %s)\n", hostedDomain, allowedHostedDomain)
			http.Error(w, "Forbidden", 403)
			return
		}

		config := fmt.Sprintf(outputTemplate, email, email, clientId, clientSecret, tokenResponse.IdToken, idpIssuerUrl, tokenResponse.RefreshToken)

		output := config

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(output))
		if err != nil {
			log.Println("failed to write about response")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

func getHostedDomain(accessToken string) (string, error) {
	uri, _ := url.Parse(userInfoUrl)
	q := uri.Query()
	q.Set("alt", "json")
	q.Set("access_token", accessToken)

	uri.RawQuery = q.Encode()
	resp, err := http.Get(uri.String())

	if err != nil {
		return "", err
	}

	hd := &HostedDomain{}

	err = json.NewDecoder(resp.Body).Decode(hd)

	return hd.HostedDomain, nil
}

func GetUserEmail(accessToken string) (string, error) {

	uri, err := url.Parse(userInfoUrl)
	if err != nil {
		log.Fatal(err)
	}

	q := uri.Query()
	q.Set("access_token", accessToken)
	uri.RawQuery = q.Encode()

	resp, _ := http.Get(uri.String())

	defer resp.Body.Close()

	userEmail := &UserEmail{}

	err = json.NewDecoder(resp.Body).Decode(userEmail)
	if err != nil {
		return "", err
	}

	return userEmail.Email, nil
}

func GetTokens(code string) (*TokenResponse, error) {

	form := url.Values{
		"code":          {code},
		"client_secret": {clientSecret},
		"redirect_uri":  {callbackUrl},
		"grant_type":    {"authorization_code"},
		"client_id":     {clientId},
	}

	resp, err := http.PostForm(tokenUrl, form)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	tr := &TokenResponse{}

	err = json.NewDecoder(resp.Body).Decode(tr)
	if err != nil {
		return nil, err
	}
	return tr, nil

}
