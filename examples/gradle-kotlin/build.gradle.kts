plugins {
	id("org.jetbrains.kotlin.jvm") version "1.7.20"
	application
	id("com.code-intelligence.cifuzz") version "1.10.0"
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
	testImplementation(platform("org.junit:junit-bom:5.10.0"))
	testImplementation("org.junit.jupiter:junit-jupiter")
}

tasks.test {
	useJUnitPlatform()
	testLogging {
		events("passed", "skipped", "failed")
	}
}

application {
	mainClass.set("MainKt")
}
