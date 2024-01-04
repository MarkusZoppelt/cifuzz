plugins {
    id("org.jetbrains.kotlin.jvm") version "1.7.20"
    id("java-library")
    id("com.code-intelligence.cifuzz") version "1.12.0"
}

repositories {
	maven {
		name = "CodeIntelligenceRepository"
		url = uri("https://gitlab.code-intelligence.com/api/v4/projects/89/packages/maven")
		credentials(PasswordCredentials::class)
		content {
			includeGroupByRegex("com\\.code-intelligence.*")
		}
	}
	mavenCentral()
}
