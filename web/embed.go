package web

import "embed"

// Assets embeds the static web files for the Total Connect web UI.
//go:embed index.html style.css app.js
var Assets embed.FS
