package main

import (
	"api/pkg/prompt"
	"api/pkg/repository"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
	r := gin.Default()

	r.POST("prompt/", CreatePrompt)

	r.GET("prompt/:namespace/:name", GetPrompt)

	ginLambda = ginadapter.New(r)
}

// validateVersion checks if the version is either "LATEST" or a valid version number.
func validateVersion(version string) bool {
	// Regular expression to match a version number like "0001.0002.0013"
	// This pattern allows for leading zeros in each numeric segment of the version.
	versionRegex := regexp.MustCompile(`^(LATEST|(\d{1,4}\.\d{1,4}\.\d{1,4}))$`)

	return versionRegex.MatchString(version)
}

func CreatePrompt(c *gin.Context) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	db := dynamodb.New(sess)

	repository := repository.NewRepository(db)

	var prompt prompt.Prompt
	err := c.BindJSON(&prompt)
	if err != nil {
		fmt.Println("Error binding JSON")
		fmt.Println(err)
		c.AbortWithError(400, err)
		return
	}

	err = repository.PutPrompt(&prompt)

	if err != nil {
		fmt.Println("Error saving prompt")
		c.AbortWithError(500, err)
		return
	}

	jsonData, err := json.Marshal(prompt)
	if err != nil {
		fmt.Println("Error marshalling prompt")
		c.AbortWithError(500, err)
		return
	}

	c.Data(
		200,
		"application/json",
		jsonData,
	)
}

const namespacePathParameterName = "namespace"
const promptPathParameterName = "name"
const versionQueryParameterName = "version"

func GetPrompt(c *gin.Context) {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	db := dynamodb.New(sess)

	repository := repository.NewRepository(db)

	organization := c.Param(namespacePathParameterName)
	name := c.Param(promptPathParameterName)

	reference := fmt.Sprintf("%s#%s", organization, name)

	// the version is a query parameter
	version := c.Query(versionQueryParameterName)

	if version == "" {
		version = "LATEST"
	}

	if !validateVersion(version) {
		c.AbortWithStatusJSON(400, gin.H{
			"error": "Invalid version format",
		},
		)
		return
	}

	prompt, err := repository.GetPrompt(reference, version)
	if prompt == nil {
		c.AbortWithStatusJSON(404, gin.H{
			"error": err.Error(),
		},
		)
		return
	}

	jsonPrompt, err := json.Marshal(prompt)

	if err != nil {
		fmt.Println("Error marshalling prompt")
		c.AbortWithError(500, err)
		return
	}

	c.Data(200, "application/json", jsonPrompt)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
