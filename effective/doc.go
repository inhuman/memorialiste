// Package effective resolves the final per-doc runtime configuration by
// merging five layers (hard-coded < manifest defaults < manifest per-doc <
// env var < CLI flag) into a single Effective struct consumed by the
// pipeline.
package effective
