# User Preference Environment Variables Solution

## Overview

This solution implements a scalable approach to expose user preferences as environment variables during Terraform operations. User preferences are automatically converted from camelCase field names to `USER_PREF_*` prefixed UPPER_SNAKE_CASE environment variables.

## Examples

The following user preferences:
```json
{
  "azureRegion": "East US",
  "userDefaultVMSize": "Standard_DS2_v2",
  "terminalAutoScroll": true
}
```

Are automatically converted to environment variables:
```bash
USER_PREF_AZURE_REGION="East US"
USER_PREF_USER_DEFAULT_VM_SIZE="Standard_DS2_v2"
USER_PREF_TERMINAL_AUTO_SCROLL="true"
```

## Implementation Details

### 1. Helper Function (`helpers.go`)

**Function:** `ConvertStructToEnvVars(v interface{}, prefix string) map[string]string`

This generic function:
- Accepts any struct and a prefix
- Converts struct field names from camelCase to UPPER_SNAKE_CASE
- Returns a map of environment variable key-value pairs
- Is fully scalable - adding new preference fields requires no code changes

**Algorithm:**
1. Marshals the struct to JSON to get actual values
2. Unmarshals to a map to access values by JSON field names
3. Uses reflection to get struct field names
4. Converts field names using the existing `CamelToConventional` helper
5. Uppercases the conventional name and adds the prefix

### 2. Repository Layer (`terraform.go`)

**Updated:** `buildUserEnvironment()` 
- Now accepts a `Preference` parameter
- Calls `helper.ConvertStructToEnvVars(preference, "USER_PREF_")`
- Merges preference environment variables with other environment variables

**Updated:** `TerraformAction()` and `ExecuteScript()`
- Both now accept a `Preference` parameter
- Pass preferences to `buildUserEnvironment()`

### 3. Entity Interface (`terraform.go`)

**Updated:** `TerraformRepository` interface
- Method signatures now include `preference Preference` parameter

### 4. Service Layer (`terraform.go`)

**Updated:** `helperTerraformAction()` 
- Already fetched user preference for VM size validation
- Now passes preference to repository's `TerraformAction()` method

**Updated:** `helperExecuteScript()`
- Now fetches user preference
- Passes preference to repository's `ExecuteScript()` method

## Benefits

1. **Scalability**: Adding new user preferences automatically creates corresponding environment variables
2. **Consistency**: All preferences follow the same naming convention
3. **Type Safety**: Uses Go's type system and reflection
4. **No Code Duplication**: Single helper function handles all conversions
5. **Testability**: Fully unit tested with comprehensive test coverage

## Usage in Terraform/Scripts

Environment variables are available in all Terraform operations and extend scripts:

```bash
# Access in bash scripts
echo "User's preferred region: $USER_PREF_AZURE_REGION"
echo "User's default VM size: $USER_PREF_USER_DEFAULT_VM_SIZE"

# Use in Terraform (if needed)
# Note: These are regular env vars, not TF_VAR_* prefixed
```

## Testing

Run tests with:
```bash
cd server
go test ./internal/helper/... -v
```

All tests pass successfully, verifying:
- Correct conversion from camelCase to UPPER_SNAKE_CASE
- Correct prefix application
- Correct value preservation
- Handling of different data types (string, bool, etc.)

## Files Modified

1. `/server/internal/helper/helpers.go` - Added `ConvertStructToEnvVars()` function
2. `/server/internal/helper/helpers_test.go` - Added comprehensive unit tests
3. `/server/internal/repository/terraform.go` - Updated to accept and use preferences
4. `/server/internal/entity/terraform.go` - Updated interface signatures
5. `/server/internal/service/terraform.go` - Updated to pass preferences to repository

## Future Enhancements

This solution is ready to handle any new preference fields automatically. To add a new preference:

1. Add the field to `entity.Preference` struct
2. Add the JSON tag
3. That's it! The environment variable will be automatically created

Example:
```go
type Preference struct {
    AzureRegion        string `json:"azureRegion"`
    UserDefaultVMSize  string `json:"userDefaultVMSize"`
    TerminalAutoScroll bool   `json:"terminalAutoScroll"`
    NewField           string `json:"newField"` // Automatically becomes USER_PREF_NEW_FIELD
}
```
