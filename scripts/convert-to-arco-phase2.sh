#!/bin/bash
# Phase 2: Replace el-icon patterns and Element Plus icon imports with Arco Design icons
# Also handles el-table → a-table, el-steps → a-steps, v-loading, tag types

KUBE_DIR="web/src/views/kubernetes"

find "$KUBE_DIR" -name "*.vue" | while read file; do
  echo "Processing icons: $file"

  # --- Replace <el-icon><IconName /></el-icon> → <icon-name /> ---
  # Common icon mappings
  sed -i \
    -e 's/<el-icon[^>]*><Platform \/><\/el-icon>/<icon-apps \/>/g' \
    -e 's/<el-icon[^>]*><CircleCheck \/><\/el-icon>/<icon-check-circle \/>/g' \
    -e 's/<el-icon[^>]*><Odometer \/><\/el-icon>/<icon-dashboard \/>/g' \
    -e 's/<el-icon[^>]*><Connection \/><\/el-icon>/<icon-link \/>/g' \
    -e 's/<el-icon[^>]*><Search \/><\/el-icon>/<icon-search \/>/g' \
    -e 's/<el-icon[^>]*><InfoFilled \/><\/el-icon>/<icon-info-circle \/>/g' \
    -e 's/<el-icon[^>]*><Refresh \/><\/el-icon>/<icon-refresh \/>/g' \
    -e 's/<el-icon[^>]*><RefreshLeft \/><\/el-icon>/<icon-undo \/>/g' \
    -e 's/<el-icon[^>]*><Plus \/><\/el-icon>/<icon-plus \/>/g' \
    -e 's/<el-icon[^>]*><Edit \/><\/el-icon>/<icon-edit \/>/g' \
    -e 's/<el-icon[^>]*><Delete \/><\/el-icon>/<icon-delete \/>/g' \
    -e 's/<el-icon[^>]*><Lock \/><\/el-icon>/<icon-lock \/>/g' \
    -e 's/<el-icon[^>]*><Key \/><\/el-icon>/<icon-safe \/>/g' \
    -e 's/<el-icon[^>]*><Upload \/><\/el-icon>/<icon-upload \/>/g' \
    -e 's/<el-icon[^>]*><Download \/><\/el-icon>/<icon-download \/>/g' \
    -e 's/<el-icon[^>]*><DocumentCopy \/><\/el-icon>/<icon-copy \/>/g' \
    -e 's/<el-icon[^>]*><Warning \/><\/el-icon>/<icon-exclamation-circle \/>/g' \
    -e 's/<el-icon[^>]*><User \/><\/el-icon>/<icon-user \/>/g' \
    -e 's/<el-icon[^>]*><Monitor \/><\/el-icon>/<icon-desktop \/>/g' \
    -e 's/<el-icon[^>]*><Cpu \/><\/el-icon>/<icon-thunderbolt \/>/g' \
    -e 's/<el-icon[^>]*><Tools \/><\/el-icon>/<icon-tool \/>/g' \
    -e 's/<el-icon[^>]*><FolderOpened \/><\/el-icon>/<icon-folder \/>/g' \
    -e 's/<el-icon[^>]*><Document \/><\/el-icon>/<icon-file \/>/g' \
    -e 's/<el-icon[^>]*><Files \/><\/el-icon>/<icon-storage \/>/g' \
    -e 's/<el-icon[^>]*><View \/><\/el-icon>/<icon-eye \/>/g' \
    -e 's/<el-icon[^>]*><DocumentChecked \/><\/el-icon>/<icon-file-search \/>/g' \
    -e 's/<el-icon[^>]*><PriceTag \/><\/el-icon>/<icon-tag \/>/g' \
    -e 's/<el-icon[^>]*><Clock \/><\/el-icon>/<icon-clock-circle \/>/g' \
    -e 's/<el-icon[^>]*><Setting \/><\/el-icon>/<icon-settings \/>/g' \
    -e 's/<el-icon[^>]*><ArrowLeft \/><\/el-icon>/<icon-left \/>/g' \
    -e 's/<el-icon[^>]*><ArrowRight \/><\/el-icon>/<icon-right \/>/g' \
    -e 's/<el-icon[^>]*><ArrowDown \/><\/el-icon>/<icon-down \/>/g' \
    -e 's/<el-icon[^>]*><ArrowUp \/><\/el-icon>/<icon-up \/>/g' \
    -e 's/<el-icon[^>]*><Close \/><\/el-icon>/<icon-close \/>/g' \
    -e 's/<el-icon[^>]*><Folder \/><\/el-icon>/<icon-folder \/>/g' \
    -e 's/<el-icon[^>]*><VideoPlay \/><\/el-icon>/<icon-play-arrow \/>/g' \
    -e 's/<el-icon[^>]*><VideoPause \/><\/el-icon>/<icon-pause-circle \/>/g' \
    -e 's/<el-icon[^>]*><Timer \/><\/el-icon>/<icon-clock-circle \/>/g' \
    -e 's/<el-icon[^>]*><Aim \/><\/el-icon>/<icon-bulseye \/>/g' \
    -e 's/<el-icon[^>]*><Box \/><\/el-icon>/<icon-storage \/>/g' \
    -e 's/<el-icon[^>]*><Grid \/><\/el-icon>/<icon-apps \/>/g' \
    -e 's/<el-icon[^>]*><Tickets \/><\/el-icon>/<icon-list \/>/g' \
    -e 's/<el-icon[^>]*><Calendar \/><\/el-icon>/<icon-calendar \/>/g' \
    -e 's/<el-icon[^>]*><MoreFilled \/><\/el-icon>/<icon-more \/>/g' \
    -e 's/<el-icon[^>]*><Expand \/><\/el-icon>/<icon-expand \/>/g' \
    -e 's/<el-icon[^>]*><CopyDocument \/><\/el-icon>/<icon-copy \/>/g' \
    -e 's/<el-icon[^>]*><OfficeBuilding \/><\/el-icon>/<icon-common \/>/g' \
    -e 's/<el-icon[^>]*><Promotion \/><\/el-icon>/<icon-send \/>/g' \
    -e 's/<el-icon[^>]*><DataLine \/><\/el-icon>/<icon-line-chart \/>/g' \
    -e 's/<el-icon[^>]*><DataBoard \/><\/el-icon>/<icon-bar-chart \/>/g' \
    -e 's/<el-icon[^>]*><Histogram \/><\/el-icon>/<icon-bar-chart \/>/g' \
    -e 's/<el-icon[^>]*><TrendCharts \/><\/el-icon>/<icon-rise \/>/g' \
    -e 's/<el-icon[^>]*><Stopwatch \/><\/el-icon>/<icon-clock-circle \/>/g' \
    -e 's/<el-icon[^>]*><Operation \/><\/el-icon>/<icon-settings \/>/g' \
    -e 's/<el-icon[^>]*><Memo \/><\/el-icon>/<icon-file \/>/g' \
    -e 's/<el-icon[^>]*><List \/><\/el-icon>/<icon-list \/>/g' \
    -e 's/<el-icon[^>]*><Finished \/><\/el-icon>/<icon-check-circle \/>/g' \
    -e 's/<el-icon[^>]*><SuccessFilled \/><\/el-icon>/<icon-check-circle-fill \/>/g' \
    -e 's/<el-icon[^>]*><WarningFilled \/><\/el-icon>/<icon-exclamation-circle-fill \/>/g' \
    -e 's/<el-icon[^>]*><CircleCloseFilled \/><\/el-icon>/<icon-close-circle-fill \/>/g' \
    -e 's/<el-icon[^>]*><CircleClose \/><\/el-icon>/<icon-close-circle \/>/g' \
    -e 's/<el-icon[^>]*><QuestionFilled \/><\/el-icon>/<icon-question-circle \/>/g' \
    -e 's/<el-icon[^>]*><Switch \/><\/el-icon>/<icon-swap \/>/g' \
    -e 's/<el-icon[^>]*><Sort \/><\/el-icon>/<icon-sort \/>/g' \
    -e 's/<el-icon[^>]*><Rank \/><\/el-icon>/<icon-sort \/>/g' \
    -e 's/<el-icon[^>]*><Position \/><\/el-icon>/<icon-location \/>/g' \
    -e 's/<el-icon[^>]*><Pointer \/><\/el-icon>/<icon-select-all \/>/g' \
    -e 's/<el-icon[^>]*><Notebook \/><\/el-icon>/<icon-book \/>/g' \
    -e 's/<el-icon[^>]*><Management \/><\/el-icon>/<icon-settings \/>/g' \
    "$file"

  # Handle remaining <el-icon> with style/class attributes that weren't caught
  # Pattern: <el-icon style="..."><IconName /></el-icon>
  # These need manual attention but let's try common patterns
  sed -i \
    -e 's/<el-icon style="margin-right: 6px;"><Refresh \/><\/el-icon>/<icon-refresh style="margin-right: 6px;" \/>/g' \
    -e 's/<el-icon style="margin-right: 6px;"><Plus \/><\/el-icon>/<icon-plus style="margin-right: 6px;" \/>/g' \
    -e 's/<el-icon style="margin-right: 4px;"><RefreshLeft \/><\/el-icon>/<icon-undo style="margin-right: 4px;" \/>/g' \
    -e 's/<el-icon style="margin-right: 4px;"><Search \/><\/el-icon>/<icon-search style="margin-right: 4px;" \/>/g' \
    -e 's/<el-icon style="margin-right: 6px;"><Download \/><\/el-icon>/<icon-download style="margin-right: 6px;" \/>/g' \
    -e 's/<el-icon style="margin-right: 6px;"><Upload \/><\/el-icon>/<icon-upload style="margin-right: 6px;" \/>/g' \
    "$file"

  # --- Replace el-table/el-table-column tags ---
  sed -i \
    -e 's/<el-table\b/<a-table/g' \
    -e 's/<\/el-table>/<\/a-table>/g' \
    -e 's/<el-table-column/<a-table-column/g' \
    -e 's/<\/el-table-column>/<\/a-table-column>/g' \
    "$file"

  # --- Replace el-steps/el-step ---
  sed -i \
    -e 's/<el-steps/<a-steps/g' \
    -e 's/<\/el-steps>/<\/a-steps>/g' \
    -e 's/<el-step /<a-step /g' \
    -e 's/<\/el-step>/<\/a-step>/g' \
    "$file"

  # --- Replace v-loading on tables ---
  sed -i 's/v-loading="loading"/:loading="loading"/g' "$file"
  sed -i 's/v-loading="tableLoading"/:loading="tableLoading"/g' "$file"

  # --- Tag type mapping ---
  sed -i \
    -e 's/\(a-tag.*\)type="success"/\1color="green"/g' \
    -e 's/\(a-tag.*\)type="danger"/\1color="red"/g' \
    -e 's/\(a-tag.*\)type="warning"/\1color="orangered"/g' \
    -e 's/\(a-tag.*\)type="info"/\1color="gray"/g' \
    "$file"

  # Also handle :type bindings for tags
  sed -i \
    -e "s/:type=\"getStatusType(/:color=\"getStatusType(/g" \
    -e "s/:type=\"getStatusColor(/:color=\"getStatusColor(/g" \
    "$file"

  # --- Replace Element Plus icon imports with Arco icon imports ---
  # Remove old imports
  sed -i "/from '@element-plus\/icons-vue'/d" "$file"

  # --- Remove remaining el-icon wrappers that have component :is ---
  # <el-icon class="type-icon"><component :is="type.icon" /></el-icon>
  sed -i 's/<el-icon class="\([^"]*\)"><component :is="\([^"]*\)" \/><\/el-icon>/<component :is="\2" class="\1" \/>/g' "$file"

  # Generic remaining el-icon cleanup
  sed -i \
    -e 's/<el-icon[^>]*><component :is="\([^"]*\)" \/><\/el-icon>/<component :is="\1" \/>/g' \
    "$file"

done

echo "Phase 2 done! Icon and table tag replacements complete."
