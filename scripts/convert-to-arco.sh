#!/bin/bash
# Batch convert Element Plus components to Arco Design in kubernetes Vue files
# This handles mechanical replacements. Complex changes (tables, icons) need manual work.

KUBE_DIR="web/src/views/kubernetes"

find "$KUBE_DIR" -name "*.vue" | while read file; do
  echo "Processing: $file"

  # --- Template component replacements ---
  # Simple tag replacements (opening and closing)
  sed -i \
    -e 's/<el-button/<a-button/g' \
    -e 's/<\/el-button>/<\/a-button>/g' \
    -e 's/<el-input\b/<a-input/g' \
    -e 's/<\/el-input>/<\/a-input>/g' \
    -e 's/<el-input-number/<a-input-number/g' \
    -e 's/<\/el-input-number>/<\/a-input-number>/g' \
    -e 's/<el-select\b/<a-select/g' \
    -e 's/<\/el-select>/<\/a-select>/g' \
    -e 's/<el-option\b/<a-option/g' \
    -e 's/<\/el-option>/<\/a-option>/g' \
    -e 's/<el-form\b/<a-form/g' \
    -e 's/<\/el-form>/<\/a-form>/g' \
    -e 's/<el-form-item/<a-form-item/g' \
    -e 's/<\/el-form-item>/<\/a-form-item>/g' \
    -e 's/<el-row/<a-row/g' \
    -e 's/<\/el-row>/<\/a-row>/g' \
    -e 's/<el-col/<a-col/g' \
    -e 's/<\/el-col>/<\/a-col>/g' \
    -e 's/<el-tag/<a-tag/g' \
    -e 's/<\/el-tag>/<\/a-tag>/g' \
    -e 's/<el-tooltip/<a-tooltip/g' \
    -e 's/<\/el-tooltip>/<\/a-tooltip>/g' \
    -e 's/<el-tabs/<a-tabs/g' \
    -e 's/<\/el-tabs>/<\/a-tabs>/g' \
    -e 's/<el-tab-pane/<a-tab-pane/g' \
    -e 's/<\/el-tab-pane>/<\/a-tab-pane>/g' \
    -e 's/<el-checkbox\b/<a-checkbox/g' \
    -e 's/<\/el-checkbox>/<\/a-checkbox>/g' \
    -e 's/<el-switch/<a-switch/g' \
    -e 's/<\/el-switch>/<\/a-switch>/g' \
    -e 's/<el-alert/<a-alert/g' \
    -e 's/<\/el-alert>/<\/a-alert>/g' \
    -e 's/<el-empty/<a-empty/g' \
    -e 's/<\/el-empty>/<\/a-empty>/g' \
    -e 's/<el-card/<a-card/g' \
    -e 's/<\/el-card>/<\/a-card>/g' \
    -e 's/<el-divider/<a-divider/g' \
    -e 's/<\/el-divider>/<\/a-divider>/g' \
    -e 's/<el-radio-group/<a-radio-group/g' \
    -e 's/<\/el-radio-group>/<\/a-radio-group>/g' \
    -e 's/<el-radio-button/<a-radio/g' \
    -e 's/<\/el-radio-button>/<\/a-radio>/g' \
    -e 's/<el-radio\b/<a-radio/g' \
    -e 's/<\/el-radio>/<\/a-radio>/g' \
    -e 's/<el-pagination/<a-pagination/g' \
    -e 's/<\/el-pagination>/<\/a-pagination>/g' \
    -e 's/<el-descriptions\b/<a-descriptions/g' \
    -e 's/<\/el-descriptions>/<\/a-descriptions>/g' \
    -e 's/<el-descriptions-item/<a-descriptions-item/g' \
    -e 's/<\/el-descriptions-item>/<\/a-descriptions-item>/g' \
    -e 's/<el-breadcrumb\b/<a-breadcrumb/g' \
    -e 's/<\/el-breadcrumb>/<\/a-breadcrumb>/g' \
    -e 's/<el-breadcrumb-item/<a-breadcrumb-item/g' \
    -e 's/<\/el-breadcrumb-item>/<\/a-breadcrumb-item>/g' \
    -e 's/<el-collapse\b/<a-collapse/g' \
    -e 's/<\/el-collapse>/<\/a-collapse>/g' \
    -e 's/<el-collapse-item/<a-collapse-item/g' \
    -e 's/<\/el-collapse-item>/<\/a-collapse-item>/g' \
    -e 's/<el-progress/<a-progress/g' \
    -e 's/<\/el-progress>/<\/a-progress>/g' \
    -e 's/<el-slider/<a-slider/g' \
    -e 's/<\/el-slider>/<\/a-slider>/g' \
    -e 's/<el-popover/<a-popover/g' \
    -e 's/<\/el-popover>/<\/a-popover>/g' \
    -e 's/<el-dropdown/<a-dropdown/g' \
    -e 's/<\/el-dropdown>/<\/a-dropdown>/g' \
    -e 's/<el-dropdown-menu/<a-doption/g' \
    -e 's/<\/el-dropdown-menu>/<\/a-doption>/g' \
    -e 's/<el-dropdown-item/<a-doption/g' \
    -e 's/<\/el-dropdown-item>/<\/a-doption>/g' \
    -e 's/<el-skeleton/<a-skeleton/g' \
    -e 's/<\/el-skeleton>/<\/a-skeleton>/g' \
    "$file"

  # --- Dialog → Modal ---
  sed -i \
    -e 's/<el-dialog/<a-modal/g' \
    -e 's/<\/el-dialog>/<\/a-modal>/g' \
    "$file"

  # --- Attribute changes ---
  # form-item: prop → field
  sed -i 's/\(a-form-item.*\) prop="/\1 field="/g' "$file"

  # descriptions: border → bordered (as boolean)
  sed -i 's/\(a-descriptions.*\) border\b/\1 :bordered="true"/g' "$file"

  # tab-pane: name → key, label → title
  sed -i \
    -e 's/\(a-tab-pane.*\) name="/\1 key="/g' \
    -e 's/\(a-tab-pane.*\) label="/\1 title="/g' \
    "$file"

  # tabs: v-model → v-model:active-key
  sed -i 's/\(a-tabs.*\)v-model="/\1v-model:active-key="/g' "$file"

  # dialog/modal: v-model → v-model:visible
  sed -i 's/\(a-modal.*\)v-model="/\1v-model:visible="/g' "$file"

  # button link → type="text"
  sed -i 's/\(a-button.*\) link\b/\1 type="text"/g' "$file"

  # pagination: page-sizes → page-size-options, layout → show-total show-page-size show-jumper
  sed -i 's/:page-sizes=/:page-size-options=/g' "$file"
  sed -i 's/layout="total, sizes, prev, pager, next, jumper"/show-total show-page-size show-jumper/g' "$file"

  # radio-button: label → value
  sed -i 's/\(a-radio.*\) label="/\1 value="/g' "$file"

  # --- Script import changes ---
  # ElMessage → Message
  sed -i "s/import { ElMessage, ElMessageBox } from 'element-plus'/import { Message, Modal } from '@arco-design\/web-vue'/g" "$file"
  sed -i "s/import { ElMessage } from 'element-plus'/import { Message } from '@arco-design\/web-vue'/g" "$file"
  sed -i "s/import type { FormInstance } from 'element-plus'//g" "$file"

  # ElMessage → Message calls
  sed -i \
    -e 's/ElMessage\.success/Message.success/g' \
    -e 's/ElMessage\.error/Message.error/g' \
    -e 's/ElMessage\.warning/Message.warning/g' \
    -e 's/ElMessage\.info/Message.info/g' \
    "$file"

  # --- Style deep selector changes ---
  sed -i \
    -e 's/:deep(\.el-/:deep(.arco-/g' \
    -e 's/\.el-table/\.arco-table/g' \
    -e 's/\.el-dialog/\.arco-modal/g' \
    -e 's/\.el-form/\.arco-form/g' \
    -e 's/\.el-input/\.arco-input/g' \
    -e 's/\.el-select/\.arco-select/g' \
    -e 's/\.el-button/\.arco-btn/g' \
    -e 's/\.el-tabs/\.arco-tabs/g' \
    -e 's/\.el-tag/\.arco-tag/g' \
    -e 's/\.el-checkbox/\.arco-checkbox/g' \
    -e 's/\.el-pagination/\.arco-pagination/g' \
    -e 's/\.el-descriptions/\.arco-descriptions/g' \
    "$file"

done

echo "Done! Mechanical replacements complete."
echo "Manual work still needed:"
echo "  1. el-table → a-table with columns array"
echo "  2. el-icon → Arco icon components"
echo "  3. ElMessageBox.confirm → Modal.confirm restructuring"
echo "  4. v-loading → :loading prop"
echo "  5. Tag type mapping (success→green, etc.)"
echo "  6. Form validation API changes"
echo "  7. Import cleanup (@element-plus/icons-vue → @arco-design/web-vue/es/icon)"
