*** Begin Patch
*** Update File: pkg/authtoken/keyring.go
@@
-import (
-    "errors"
-    "github.com/zalando/go-keyring"
-)
+import (
+    "errors"
+    "fmt"
+    "github.com/zalando/go-keyring"
+)
+
+// ErrKeyringFailure represents an operational failure of the keyring service
+// (e.g., locked keychain, missing daemon). It is distinct from ErrNotFound.
+var ErrKeyringFailure = fmt.Errorf("keyring access error")
@@
-func Get(host string) (string, error) {
-    token, err := keyring.Get("github.com", host)
-    if err != nil {
-        if errors.Is(err, keyring.ErrNotFound) {
-            return "", keyring.ErrNotFound
-        }
-        return "", err
-    }
-    return token, nil
-}
+// Get retrieves a token for the given host from the system keyring.
+// It returns ErrNotFound when no secret is stored for that host.
+// For any other error from the keyring backend, it returns ErrKeyringFailure
+// wrapping the underlying error.
+func Get(host string) (string, error) {
+    token, err := keyring.Get("github.com", host)
+    if err != nil {
+        if errors.Is(err, keyring.ErrNotFound) {
+            return "", keyring.ErrNotFound
+        }
+        // Wrap the actual keyring error so callers can distinguish operational failures.
+        return "", fmt.Errorf("%w: %w", ErrKeyringFailure, err)
+    }
+    return token, nil
+}
*** End Patch
*** End Patch