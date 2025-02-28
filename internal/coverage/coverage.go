package coverage

import "code-intelligence.com/cifuzz/internal/config"

const FormatHTML = "html"
const FormatLCOV = "lcov"
const FormatJacocoXML = "jacocoxml"

var ValidOutputFormats = map[string][]string{
	config.BuildSystemCMake:  {FormatHTML, FormatLCOV},
	config.BuildSystemBazel:  {FormatHTML, FormatLCOV},
	config.BuildSystemOther:  {FormatHTML, FormatLCOV},
	config.BuildSystemMaven:  {FormatHTML, FormatLCOV, FormatJacocoXML},
	config.BuildSystemGradle: {FormatHTML, FormatLCOV, FormatJacocoXML},
	config.BuildSystemNodeJS: {FormatHTML, FormatLCOV},
}
