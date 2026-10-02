package renderassets

import _ "embed"

//go:embed DejaVuSans.ttf
var TextFont []byte

//go:embed twitch-icon.png
var TwitchIcon []byte
