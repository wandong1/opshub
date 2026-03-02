package inspection

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	biz "github.com/ydcloud-dy/opshub/internal/biz/inspection"
	"gorm.io/gorm"
)

type probeVariableRepo struct {
	db            *gorm.DB
	encryptionKey []byte
}

func NewProbeVariableRepo(db *gorm.DB) biz.ProbeVariableRepo {
	encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")
	return &probeVariableRepo{db: db, encryptionKey: encryptionKey}
}

func (r *probeVariableRepo) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (r *probeVariableRepo) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (r *probeVariableRepo) Create(ctx context.Context, v *biz.ProbeVariable) error {
	if v.VarType == biz.VariableTypeSecret {
		encrypted, err := r.encrypt(v.Value)
		if err != nil {
			return fmt.Errorf("encrypt value: %w", err)
		}
		v.Value = encrypted
	}
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *probeVariableRepo) Update(ctx context.Context, v *biz.ProbeVariable) error {
	if v.VarType == biz.VariableTypeSecret && v.Value != "" {
		encrypted, err := r.encrypt(v.Value)
		if err != nil {
			return fmt.Errorf("encrypt value: %w", err)
		}
		v.Value = encrypted
	}
	return r.db.WithContext(ctx).Model(v).Omit("created_at").Select("*").Updates(v).Error
}

func (r *probeVariableRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&biz.ProbeVariable{}, id).Error
}

func (r *probeVariableRepo) GetByID(ctx context.Context, id uint) (*biz.ProbeVariable, error) {
	var v biz.ProbeVariable
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, err
	}
	if v.VarType == biz.VariableTypeSecret {
		decrypted, err := r.decrypt(v.Value)
		if err == nil {
			v.Value = decrypted
		}
	}
	return &v, nil
}

func (r *probeVariableRepo) List(ctx context.Context, page, pageSize int, keyword, varType, groupIDs string) ([]*biz.ProbeVariable, int64, error) {
	var vars []*biz.ProbeVariable
	var total int64
	query := r.db.WithContext(ctx).Model(&biz.ProbeVariable{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if varType != "" {
		query = query.Where("var_type = ?", varType)
	}
	if groupIDs != "" {
		// Filter: show variables that are ungrouped OR have intersection with requested groups
		conditions := []string{"group_ids = ''"}
		for _, s := range strings.Split(groupIDs, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				conditions = append(conditions, fmt.Sprintf("FIND_IN_SET('%s', group_ids) > 0", s))
			}
		}
		query = query.Where(strings.Join(conditions, " OR "))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&vars).Error; err != nil {
		return nil, 0, err
	}
	// Mask secret values
	for _, v := range vars {
		if v.VarType == biz.VariableTypeSecret {
			v.Value = "******"
		}
	}
	return vars, total, nil
}

func (r *probeVariableRepo) GetByNames(ctx context.Context, names []string, allowedGroupIDs []uint) ([]*biz.ProbeVariable, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var vars []*biz.ProbeVariable
	if err := r.db.WithContext(ctx).Where("name IN ?", names).Find(&vars).Error; err != nil {
		return nil, err
	}
	// Filter by group scope:
	// - If probe has no groups (allowedGroupIDs empty): only return ungrouped variables (GroupIDs == "")
	// - If probe has groups: return ungrouped variables + variables whose groups intersect
	// When multiple variables share the same name, grouped ones take priority over ungrouped ones
	ungrouped := make(map[string]*biz.ProbeVariable)  // name -> var (GroupIDs == "")
	grouped := make(map[string]*biz.ProbeVariable)     // name -> var (matched group)
	for _, v := range vars {
		if v.GroupIDs == "" {
			ungrouped[v.Name] = v
		} else if len(allowedGroupIDs) > 0 && hasGroupIntersection(v.GroupIDs, allowedGroupIDs) {
			grouped[v.Name] = v
		}
	}
	result := make([]*biz.ProbeVariable, 0, len(names))
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		// Grouped variable takes priority
		if v, ok := grouped[name]; ok {
			decrypted := r.decryptIfSecret(v)
			result = append(result, decrypted)
		} else if v, ok := ungrouped[name]; ok {
			decrypted := r.decryptIfSecret(v)
			result = append(result, decrypted)
		}
	}
	return result, nil
}

func (r *probeVariableRepo) decryptIfSecret(v *biz.ProbeVariable) *biz.ProbeVariable {
	if v.VarType == biz.VariableTypeSecret {
		decrypted, err := r.decrypt(v.Value)
		if err == nil {
			v.Value = decrypted
		}
	}
	return v
}

func hasGroupIntersection(groupIDsStr string, allowedIDs []uint) bool {
	allowed := make(map[uint]bool, len(allowedIDs))
	for _, id := range allowedIDs {
		allowed[id] = true
	}
	for _, s := range strings.Split(groupIDsStr, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.ParseUint(s, 10, 64); err == nil {
			if allowed[uint(id)] {
				return true
			}
		}
	}
	return false
}
