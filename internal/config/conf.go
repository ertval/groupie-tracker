package config

// WithCache controls whether the repository should cache artist images locally.
// If true, the repository will attempt to download images into the local
// `static/img/artists` cache during data loading. If false, images will be
// left as provided by the API (no caching attempted).
//
// Default: true (preserves previous behavior).
var WithCache = true
