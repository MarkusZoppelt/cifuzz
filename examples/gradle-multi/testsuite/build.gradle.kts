plugins {
	id("java-library")
	id("org.jetbrains.kotlin.jvm") version "1.7.20"
	id("com.code-intelligence.cifuzz") version "1.12.0"
}

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
	mavenCentral()
}

dependencies {
	implementation(project(":app"))
}
