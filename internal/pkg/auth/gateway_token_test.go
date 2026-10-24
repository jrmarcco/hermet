package auth

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// 测试用签发参数 ( 与生产契约一致 )。
const (
	testIssuer   = "synp-auth"
	testAudience = "synp-gateway"
	testJti      = "test-jti"
)

// genTestKeyPair 使用 openssl 生成 RSA 测试密钥对 ( 与 scripts/gen_jwt_rsa_keys.sh 同参数 )。
func genTestKeyPair(t *testing.T) (privatePEM, publicPEM string) {
	t.Helper()

	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")

	cmds := [][]string{
		{"openssl", "genpkey", "-algorithm", "RSA", "-pkeyopt", "rsa_keygen_bits:2048", "-out", privatePath},
		{"openssl", "pkey", "-in", privatePath, "-pubout", "-out", publicPath},
	}
	for _, args := range cmds {
		// 固定的 openssl 命令模板，参数来自上面的字面量。
		out, err := exec.CommandContext(t.Context(), args[0], args[1:]...).CombinedOutput() //nolint:gosec // G204: 测试内的固定 openssl 命令
		require.NoError(t, err, "openssl %v failed: %s", args, out)
	}

	privateKeyPEM, err := os.ReadFile(privatePath) //nolint:gosec // G304: 路径来自 t.TempDir()，非外部输入
	require.NoError(t, err)
	publicKeyPEM, err := os.ReadFile(publicPath) //nolint:gosec // G304: 路径来自 t.TempDir()，非外部输入
	require.NoError(t, err)

	return string(privateKeyPEM), string(publicKeyPEM)
}

func newTestSigner(t *testing.T, privatePEM, publicPEM string) *RS256Signer {
	t.Helper()

	signer, err := NewRS256Signer(
		SignerConfig{
			PrivatePEM: privatePEM,
			PublicPEM:  publicPEM,

			// kid 在测试中仅作标识用 ( 生产侧为 DER 公钥 SHA-256 前 16 位 hex )。
			Kid: "test-kid",

			Issuer:     testIssuer,
			Audience:   testAudience,
			Expiration: time.Hour,
		},
		func() string { return testJti },
	)
	require.NoError(t, err)
	return signer
}

func TestGatewayTokenRoundTrip(t *testing.T) {
	t.Parallel()

	privatePEM, publicPEM := genTestKeyPair(t)
	signer := newTestSigner(t, privatePEM, publicPEM)

	const (
		businessID = uint64(1)
		userID     = uint64(42)
		sessionID  = "session-1"
	)

	token, err := signer.Sign(businessID, userID, sessionID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := signer.Verifier().Verify(token)
	require.NoError(t, err)

	require.Equal(t, businessID, claims.BusinessID)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, sessionID, claims.SessionID)

	require.Equal(t, testIssuer, claims.Issuer)
	require.Equal(t, jwt.ClaimStrings{testAudience}, claims.Audience)
	require.Equal(t, testJti, claims.ID)
	require.NotNil(t, claims.ExpiresAt)
}

func TestGatewayTokenTampered(t *testing.T) {
	t.Parallel()

	_, publicPEM := genTestKeyPair(t)

	// 用另一对密钥签发，验证公钥不匹配时验签失败。
	otherPrivate, _ := genTestKeyPair(t)
	otherSigner := newTestSigner(t, otherPrivate, publicPEM)

	token, err := otherSigner.Sign(1, 1, "session-1")
	require.NoError(t, err)

	verifier, err := NewRS256Verifier(publicPEM)
	require.NoError(t, err)

	_, err = verifier.Verify(token)
	require.Error(t, err)
}

func TestBuildJWKS(t *testing.T) {
	t.Parallel()

	_, publicPEM := genTestKeyPair(t)

	jwks, err := BuildJWKS("test-kid", publicPEM)
	require.NoError(t, err)
	require.Len(t, jwks.Keys, 1)

	key := jwks.Keys[0]
	require.Equal(t, "RSA", key.Kty)
	require.Equal(t, "test-kid", key.Kid)
	require.Equal(t, "sig", key.Use)
	require.Equal(t, "RS256", key.Alg)
	require.NotEmpty(t, key.N)
	require.NotEmpty(t, key.E)

	body, err := jwks.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(body), `"kid":"test-kid"`)

	_, err = BuildJWKS("", publicPEM)
	require.Error(t, err)
}
