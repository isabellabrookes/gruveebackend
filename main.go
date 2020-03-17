package main

// creativenobu - "compiled but feels interpreted (02/26/20)
// pheonix_d123 - "Felt Compiled. Might interpret later" (02/26/20)
// sillyonly - "YOU ALWAYS CLEAN MASTER BY FORCE PUSHING THE PERFECT CODE AND NOT THE CODE YOU WROTE" (02/23/20)
// sillyonly - "OR PUSH AFTER AN APPROVED PR" (02/23/20)
// no_neon_one - "have you tried Flutter?" (02/26/20)
// MrDemonWolf - "A Furry was here OwO" (03/08/20)
import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/joho/godotenv"
	"github.com/pixelogicdev/gruveebackend/cmd/createuser"
	"github.com/pixelogicdev/gruveebackend/cmd/socialplatform"
	"github.com/pixelogicdev/gruveebackend/cmd/socialtokenrefresh"
	"github.com/pixelogicdev/gruveebackend/cmd/spotifyauth"
	"github.com/pixelogicdev/gruveebackend/cmd/tokengen"
)

// InukApp - "Swift > Go" (03/15/20)
func main() {
	// Load in ENV file
	goEnvErr := godotenv.Load("./internal/config.yaml")
	if goEnvErr != nil {
		log.Printf("SocialTokenRefresh [Load GoEnv]: %v\n", goEnvErr)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	funcframework.RegisterHTTPFunction("/authorizeWithSpotify", spotifyauth.AuthorizeWithSpotify)
	funcframework.RegisterHTTPFunction("/generateToken", tokengen.GenerateCustomToken)
	funcframework.RegisterHTTPFunction("/createSocialPlatform", socialplatform.CreateSocialPlatform)
	funcframework.RegisterHTTPFunction("/createUser", createuser.CreateUser)
	funcframework.RegisterHTTPFunction("/socialTokenRefresh", socialtokenrefresh.SocialTokenRefresh)
	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
