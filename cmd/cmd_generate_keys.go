package cmd

import (
	"bufio"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type KeyPairs struct {
	PrivateKeyPath string
	PublicKeyPath  string
}

var filePrefix string

// 指定文本 随机数种子
var cmdSeed string

// deterministicReader 是一个确定性随机源，用于测试
type deterministicReader struct {
	seed  int64
	index int64
}

// generateKeyWithDeterministicSeed 使用确定性种子生成 RSA 密钥（仅用于测试）
// ⚠️ 安全警告：输出可重现，绝不能用于生产环境
func generateKeyWithDeterministicSeed(seed int64) (*rsa.PrivateKey, error) {
	reader := &deterministicReader{seed: seed, index: 0}
	return rsa.GenerateKey(reader, 2048)
}

func (r *deterministicReader) Read(p []byte) (n int, err error) {
	h := sha256.New()
	for i := range p {
		h.Reset()
		h.Write([]byte(fmt.Sprintf("%d-%d", r.seed, r.index)))
		sum := h.Sum(nil)
		p[i] = sum[0]
		r.index++
	}
	return len(p), nil
}

// ❌ 仅用于测试！不要用于生产！
func generateKeyWithSeed(seed int64) (*rsa.PrivateKey, error) {
	reader := &deterministicReader{seed: seed}
	return rsa.GenerateKey(reader, 2048)
}

func generateKeyWithSeedString(seedStr string) (*rsa.PrivateKey, error) {
	reader := NewHMACReader(seedStr)
	return rsa.GenerateKey(reader, 2048)
}

// HMACReader 是一个确定性随机字节流生成器
// 使用 HMAC-SHA256 作为核心 PRF（伪随机函数）
type HMACReader struct {
	key     []byte // HMAC 密钥（来自输入字符串）
	counter uint64 // 计数器，确保每次输出不同
}

// NewHMACReader 创建一个新的确定性随机源
// seed: 任意字符串，如 "guoshanshan"
func NewHMACReader(seed string) io.Reader {
	return &HMACReader{
		key:     []byte(seed),
		counter: 0,
	}
}

func (r *HMACReader) Read(p []byte) (n int, err error) {
	h := hmac.New(sha256.New, r.key)
	buf := make([]byte, 8) // 用于编码 counter

	for i := 0; i < len(p); i++ {
		// 将当前 counter 写入 buffer
		binary.BigEndian.PutUint64(buf, r.counter)

		// HMAC(key, counter) → 32 字节输出
		h.Write(buf)
		sum := h.Sum(nil)

		// 取第一个字节填充输出
		p[i] = sum[0]

		// 递增 counter
		r.counter++

		// 重置 HMAC 用于下一次
		h.Reset()
	}

	return len(p), nil
}

// findHomonymKeyFile 检查指定前缀的密钥文件是否已存在
// 返回：*KeyPairs（路径信息），exists（是否存在），error（仅返回系统错误）
func findHomonymKeyFile(prefix string) (*KeyPairs, bool, error) {
	// 确定前缀
	pFix := strings.TrimSpace(prefix)
	if pFix == "" {
		username := getWinUserName()
		username = filepath.Base(username)
		pFix = username
	}

	pubPath := "./" + pFix + "_public_key.pem"
	privPath := "./" + pFix + "_private_key.pem"

	pubExists := false
	privExists := false

	// 检查公钥文件
	if pubFileInfo, err := os.Stat(pubPath); err == nil {
		pubExists = true
		pubPath, err = filepath.Abs(pubFileInfo.Name())
		if err != nil {
			return nil, false, fmt.Errorf("检查公钥文件时发生系统错误: %w", err)
		}
	} else if !os.IsNotExist(err) {
		// 真正的错误（如权限问题）
		return nil, false, fmt.Errorf("检查公钥文件时发生系统错误: %w", err)
	}

	// 检查私钥文件
	if privFileInfo, err := os.Stat(privPath); err == nil {
		privExists = true
		file, err := os.Open(privFileInfo.Name())
		if err != nil {
			return nil, false, fmt.Errorf("检查私钥文件时发生系统错误: %w", err)
		}
		privPath, err = filepath.Abs(file.Name())
		if err != nil {
			return nil, false, fmt.Errorf("检查私钥文件时发生系统错误: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("检查私钥文件时发生系统错误: %w", err)
	}

	// 只有当两个文件都存在时，才认为“同名密钥对已存在”
	if pubExists && privExists {
		logger.Infof("找到已存在的公钥文件: %s", pubPath)
		logger.Infof("找到已存在的私钥文件: %s", privPath)
		return &KeyPairs{
			PublicKeyPath:  pubPath,
			PrivateKeyPath: privPath,
		}, true, nil
	}

	// 文件不存在，不是错误，只是状态
	return nil, false, nil
}

// genKeys 生成 RSA 密钥对
// prefix: 文件前缀
// seedStr: 可选种子（仅用于测试），为空则使用安全随机
func genKeys(prefix string, seedStr string) (*KeyPairs, error) {
	// 1. 检查是否已存在同名文件
	keyPairs, exists, err := findHomonymKeyFile(prefix)
	if err != nil {
		return nil, fmt.Errorf("检查密钥文件冲突时出错: %w", err)
	}

	if exists {
		logger.Infof("警告：已存在同名密钥文件，继续将覆盖文件:")
		logger.Infof("  - %s", keyPairs.PrivateKeyPath)
		logger.Infof("  - %s", keyPairs.PublicKeyPath)
		logger.Infof("输入 1 继续，其他任意键取消:")

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan(); scanner.Text() != "1" {
			logger.Infof("操作已取消。")
			return nil, fmt.Errorf("用户取消操作")
		}
	}

	// 2. 确定文件名
	username := filepath.Base(getWinUserName()) // 假设 getWinUserName() 已定义
	if prefix != "" {
		username = prefix
	}
	keysPath := filepath.Join(filepath.Dir(GetConfigManager().configPath), "keys")

	_, err = os.Stat(keysPath)
	if os.IsNotExist(err) {
		// 目录不存在，创建它
		err = os.MkdirAll(keysPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("创建密钥目录失败: %w", err)
		}
	}

	privKeyPrefix := filepath.Join(keysPath, username+"_private_key.pem")
	pubKeyPrefix := filepath.Join(keysPath, username+"_public_key.pem")

	// 3. 生成私钥
	var privateKey *rsa.PrivateKey
	if seedStr != "" {
		logger.Infof("使用种子 [%s] 生成可重现密钥（仅用于测试！）", seedStr)
		// seed, err := strconv.ParseInt(string(seedStr), 10, 64) // 修正：基数应为 10
		// if err != nil {
		// 	return nil, fmt.Errorf("解析随机数种子失败: %w", err)
		// }
		var errGen error
		privateKey, errGen = generateKeyWithSeedString(seedStr)
		if errGen != nil {
			return nil, fmt.Errorf("使用种子生成密钥失败: %w", errGen)
		}
	} else {
		logger.Infof("使用系统安全随机源生成密钥")
		var errGen error
		privateKey, errGen = rsa.GenerateKey(rand.Reader, 2048)
		if errGen != nil {
			return nil, fmt.Errorf("生成 RSA 密钥失败: %w", errGen)
		}
	}

	// 4. 保存私钥
	if err := savePrivateKey(privateKey, privKeyPrefix); err != nil {
		return nil, fmt.Errorf("保存私钥失败: %w", err)
	}

	// 5. 保存公钥
	if err := savePublicKey(&privateKey.PublicKey, pubKeyPrefix); err != nil {
		return nil, fmt.Errorf("保存公钥失败: %w", err)
	}

	// 6. 返回结果
	keyPairs = &KeyPairs{
		PrivateKeyPath: privKeyPrefix,
		PublicKeyPath:  pubKeyPrefix,
	}

	logger.Infof("密钥对生成成功:")
	logger.Infof("  - 私钥: %s", privKeyPrefix)
	logger.Infof("  - 公钥: %s", pubKeyPrefix)
	logger.Warnf("⚠️ 安全提醒: 请务必备份并安全保存您的私钥文件！")
	logger.Warnf("⚠️ 私钥是您解密数据的唯一凭证，一旦丢失将无法恢复加密数据！")
	logger.Warnf("⚠️ 建议将私钥文件保存在安全的离线存储设备中，并牢记其存放位置。")

	return keyPairs, nil
}

// savePrivateKey 保存私钥到 PEM 文件
func savePrivateKey(privateKey *rsa.PrivateKey, filename string) error {
	logger.Infof("保存私钥到: %s", filename)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建私钥文件失败: %w", err)
	}
	defer file.Close()

	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	return pem.Encode(file, pemBlock)
}

// savePublicKey 保存公钥到 PEM 文件
func savePublicKey(pubKey *rsa.PublicKey, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建公钥文件失败: %w", err)
	}
	defer file.Close()

	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("序列化公钥失败: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	return pem.Encode(file, pemBlock)
}

var generateKeysCmd = &cobra.Command{
	Use:   "gen-keys",
	Short: "Generate a RSA key pair",
	Long:  `Generates a RSA key pair and saves them as [user]_private_key.pem and [user]_public_key.pem.`,
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		keyPairs, err := genKeys(filePrefix, cmdSeed)
		if err != nil {
			logger.Fatal("Failed to generate keys:", err)
			return
		}
		logger.Infof("Key pair path:\nPrivate Key: %s\nPublic Key: %s", keyPairs.PrivateKeyPath, keyPairs.PublicKeyPath)
		logger.Warnf("⚠️ Security Reminder: Please backup and securely save your private key file!")
		logger.Warnf("⚠️ The private key is your only credential for decrypting data - if lost, encrypted data cannot be recovered!")
		logger.Warnf("⚠️ Recommended to store the private key file on secure offline storage and remember its location.")

		// 自动更新配置文件中的加密设置
		configManager := GetConfigManager()
		if configManager != nil && configManager.IsConfigLoaded() {
			err := configManager.UpdateConfig(func(config *UserConfig) {
				config.Encryption.Enabled = true
				config.Encryption.PublicKeyPath = keyPairs.PublicKeyPath
				config.Encryption.PrivateKeyPath = keyPairs.PrivateKeyPath
				config.Encryption.DefaultAlgorithm = "RSA-2048"
			})
			if err != nil {
				logger.Warnf("更新加密配置失败: %v", err)
			} else {
				logger.Info("加密配置已自动更新")
			}
		}
	},
}

func init() {
	generateKeysCmd.Flags().StringVarP(&filePrefix, "file", "f", "", "Specify a custom file prefix or path for the generated keys")
	generateKeysCmd.Flags().StringVarP(&cmdSeed, "prompt", "p", "", "Provide a random seed for the generated keys")

	rootCmd.AddCommand(generateKeysCmd)
}
