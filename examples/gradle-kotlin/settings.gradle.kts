pluginManagement {
	repositories {
		// Configure access to CI repository
		maven {
			name = "CodeIntelligenceRepository"
			url = uri("https://gitlab.code-intelligence.com/api/v4/projects/89/packages/maven")
			credentials(PasswordCredentials::class)
			content {
				includeGroupByRegex("com\\.code-intelligence.*")
			}
		}
		gradlePluginPortal()
	}
}
