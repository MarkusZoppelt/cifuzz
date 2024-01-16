plugins {
    id("org.jetbrains.kotlin.jvm") version "1.7.20"
    id("java-library")
    id("com.code-intelligence.cifuzz") version "1.13.0"
}

repositories {
	maven {
		name = "CodeIntelligenceRepository"
		url = uri("https://gitlab.code-intelligence.com/api/v4/projects/89/packages/maven")
		credentials {
			username = extra["CodeIntelligenceRepositoryUsername"].toString()
			password = extra["CodeIntelligenceRepositoryPassword"].toString()
		}
		content {
			includeGroupByRegex("com\\.code-intelligence.*")
		}
	}
	mavenCentral()
}

sourceSets.getByName("test") {
    java.srcDir("fuzzTests")
}
