package secureupdate

// UpdatePublicKeyBase64 is the Ed25519 public key used to verify detached
// signatures on Habo Client updater payloads. The matching private signing key
// is never stored in this repository and must only exist as a protected release
// secret.
const UpdatePublicKeyBase64 = "8EieXkARx1MVEUmPwAFA9uIunzxZKk3l5W1lTJFBptM="
