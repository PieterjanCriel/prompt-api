package repository

import (
	"fmt"
	"os"
	"strings"

	"api/pkg/prompt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

type NewPrompt struct {
	prompt.Prompt
	Reference  string `json:"reference"`
	Versioning string `json:"versioning"`
}

// DynamoDBAPI defines the set of DynamoDB operations used by the repository
type DynamoDBAPI interface {
	Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	TransactWriteItems(input *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
}

type Repository struct {
	db DynamoDBAPI // Use the interface
}

func NewRepository(db DynamoDBAPI) *Repository {
	return &Repository{db: db}
}

func encodeVersion(version string) string {
	parts := strings.Split(version, ".")
	for i, part := range parts {
		// Pad the part with leading zeros until its length is 4.
		parts[i] = fmt.Sprintf("%04s", part)
	}
	return strings.Join(parts, ".")
}

func decodeVersion(encodedVersion string) string {
	parts := strings.Split(encodedVersion, ".")
	for i, part := range parts {
		// Trim leading zeros from each part.
		parts[i] = strings.TrimLeft(part, "0")
		// If the part becomes an empty string after trimming, it means it was "0000", so we replace it with "0".
		if parts[i] == "" {
			parts[i] = "0"
		}
	}
	return strings.Join(parts, ".")
}

// func to compare two encoded versions and return if the first is greater than the second (these are strings with dots)
func greaterVersion(v1 string, v2 string) bool {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1); i++ {
		// if the first part of v1 is greater than the first part of v2, return true
		if parts1[i] > parts2[i] {
			return true
		}
		// if the first part of v1 is less than the first part of v2, return false
		if parts1[i] < parts2[i] {
			return false
		}
		// if the first part of v1 is equal to the first part of v2, continue to the next part
	}
	// if all parts are equal, return false
	return false
}

func (r *Repository) GetPrompt(reference, version string) (*prompt.Prompt, error) {
	// Define the key condition expression
	keyCond := expression.Key("reference").Equal(expression.Value(reference))
	if version == "" {
		version = "LATEST" // Assuming "LATEST" is used to mark the latest version
	} else {
		version = encodeVersion(version)
	}
	keyCond = keyCond.And(expression.Key("versioning").Equal(expression.Value(version)))

	// Create the expression
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// get table name from environment variable
	tableName := os.Getenv("PROMPT_TABLE_NAME")

	// Perform the query
	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(tableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	}

	result, err := r.db.Query(queryInput)
	if err != nil || len(result.Items) == 0 {
		return nil, fmt.Errorf("failed to query prompt or not found: %w", err)
	}

	// Unmarshal the result
	prompt := &prompt.Prompt{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal DynamoDB item to prompt: %w", err)
	}

	// Decode the version number
	prompt.Version = decodeVersion(prompt.Version)

	return prompt, nil
}

func (r *Repository) PutPrompt(prompt *prompt.Prompt) error {
	// Encode the version number
	prompt.Version = encodeVersion(prompt.Version)

	// get table name from environment variable
	tableName := os.Getenv("PROMPT_TABLE_NAME")

	// Current LATEST version if reference exists
	currentPrompt, err := r.GetPrompt(fmt.Sprintf("%s#%s", prompt.Namespace, prompt.Name), "LATEST")

	if currentPrompt != nil && err == nil {
		// Encode the version number
		currentPrompt.Version = encodeVersion(currentPrompt.Version)

		// If the current version is greater than the new version, return an error
		if greaterVersion(currentPrompt.Version, prompt.Version) {
			return fmt.Errorf("version %s is not greater than the current version %s", prompt.Version, currentPrompt.Version)
		} else if currentPrompt.Version == prompt.Version {
			return fmt.Errorf("version %s already exists", prompt.Version)
		}
	}

	versionedPrompt := NewPrompt{
		Prompt:     *prompt,
		Reference:  fmt.Sprintf("%s#%s", prompt.Namespace, prompt.Name),
		Versioning: prompt.Version,
	}

	latestPrompt := NewPrompt{
		Prompt:     *prompt,
		Reference:  fmt.Sprintf("%s#%s", prompt.Namespace, prompt.Name),
		Versioning: "LATEST",
	}

	// Marshal both prompts
	versionedPromptMap, err := dynamodbattribute.MarshalMap(versionedPrompt)
	if err != nil {
		return fmt.Errorf("failed to marshal versioned prompt to DynamoDB attribute: %w", err)
	}

	latestPromptMap, err := dynamodbattribute.MarshalMap(latestPrompt)
	if err != nil {
		return fmt.Errorf("failed to marshal latest prompt to DynamoDB attribute: %w", err)
	}

	// Prepare transaction input to atomically update both entries
	transactInput := &dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Put: &dynamodb.Put{
					TableName: aws.String(tableName),
					Item:      versionedPromptMap,
				},
			},
			{
				Put: &dynamodb.Put{
					TableName: aws.String(tableName),
					Item:      latestPromptMap,
				},
			},
		},
	}

	_, err = r.db.TransactWriteItems(transactInput)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}
