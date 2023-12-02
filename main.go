package main

import (
	"crypto/ed25519"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/go-fed/httpsig"
	"github.com/joho/godotenv"
	"github.com/woodpecker-ci/woodpecker/server/model"
)

type config struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type incoming struct {
	Repo          *model.Repo     `json:"repo"`
	Build         *model.Pipeline `json:"pipeline"`
	Configuration []*config       `json:"configs"`
}

//go:embed central-pipeline-config.yml
var overrideConfiguration string

func main() {
	log.Println("Woodpecker central config server")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	pubKeyPath := os.Getenv("CONFIG_SERVICE_PUBLIC_KEY_FILE") // Key in format of the one fetched from http(s)://your-woodpecker-server/api/signature/public-key
	host := os.Getenv("CONFIG_SERVICE_HOST")
	filterRegex := os.Getenv("CONFIG_SERVICE_OVERRIDE_FILTER")

	if pubKeyPath == "" && host == "" {
		log.Fatal("Please make sure CONFIG_SERVICE_HOST and CONFIG_SERVICE_PUBLIC_KEY_FILE are set properly")
	}

	pubKeyRaw, err := os.ReadFile(pubKeyPath)
	if err != nil {
		log.Fatal("Failed to read public key file")
	}

	pemblock, _ := pem.Decode(pubKeyRaw)

	b, err := x509.ParsePKIXPublicKey(pemblock.Bytes)
	if err != nil {
		log.Fatal("Failed to parse public key file ", err)
	}
	pubKey, ok := b.(ed25519.PublicKey)
	if !ok {
		log.Fatal("Failed to parse public key file")
	}

	filter := regexp.MustCompile(filterRegex)

	http.HandleFunc("/ciconfig", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// check signature
		pubKeyID := "woodpecker-ci-plugins"

		verifier, err := httpsig.NewVerifier(r)
		if err != nil {
			log.Printf("config: invalid or missing signature in http.Request")
			http.Error(w, "Invalid or Missing Signature", http.StatusBadRequest)
			return
		}

		keyID := verifier.KeyId()
		if keyID != pubKeyID {
			log.Printf("config: invalid signature in http.Request, keyID missmatch")
			http.Error(w, "Invalid Signature", http.StatusBadRequest)
			return
		}

		if err := verifier.Verify(pubKey, httpsig.ED25519); err != nil {
			log.Printf("config: invalid signature in http.Request")
			http.Error(w, "Invalid Signature", http.StatusBadRequest)
			return
		}

		var req incoming
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		err = json.Unmarshal(body, &req)
		if err != nil {
			http.Error(w, "Failed to parse JSON"+err.Error(), http.StatusBadRequest)
			return
		}

		if filter.MatchString(req.Repo.Name) {
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(map[string]interface{}{"configs": []config{
				{
					Name: "central pipe",
					Data: overrideConfiguration,
				},
			}})
			if err != nil {
				log.Printf("Error on encoding json %v\n", err)
			}
		} else {
			w.WriteHeader(http.StatusNoContent) // use default config
			// No need to write a response body
		}
	})

	err = http.ListenAndServe(os.Getenv("CONFIG_SERVICE_HOST"), nil)
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}
}
