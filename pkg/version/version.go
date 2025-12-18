package version

var (
	// EDIT THESE VALUES AFTER CREATING THE REPO FROM TEMPLATE
	// -----------------------------------------------------

	// AppName  is the CamelCase name of your app (e.g., "User", "Product")
	AppName = "todoApp"

	// GoPackage  is the name of your main service go package (e.g., "user", "product")
	// should be: all lowercase, short no hyphens, no underscores, no camelCase, usually one word
	GoPackage = "todo"

	// ServiceName is the name of your main entity/service first letter Capital (e.g., "User", "Product")
	ServiceName = "Todo"

	// DbSchemaName is the name of your main entity/service database schema can be the same as go package
	DbSchemaName = "todo"

	// AppNameKebab is the kebab-case version for your github repository (e.g., "user", "product")
	AppNameKebab = "go-cloud-k8s-todo"

	// AppNameSnake is the snake-case version for database or directory (e.g., "user", "product")
	AppNameSnake = "todo"

	// Repository is the GitHub repo
	Repository = "github.com/lao-tseu-is-alive/go-cloud-k8s-todo"

	// Version starting point
	Version = "0.0.1"

	// Revision is auto-filled by build (do not edit manually)
	Revision = "unknown"
	// BuildStamp is auto-filled by build (do not edit manually)
	BuildStamp = "unknown"
)
