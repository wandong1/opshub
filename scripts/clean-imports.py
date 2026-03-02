#!/usr/bin/env python3
"""Remove orphaned Element Plus icon import blocks from Vue files."""
import re
import glob
import os

KUBE_DIR = "/home/wandong/opus_project/src/opshub/web/src/views/kubernetes"

# Known Element Plus icon names
ICON_NAMES = {
    'Search', 'InfoFilled', 'Connection', 'Upload', 'Platform', 'Key', 'Refresh',
    'RefreshLeft', 'Plus', 'Edit', 'Delete', 'Lock', 'Download', 'Warning',
    'Monitor', 'Cpu', 'Tools', 'FolderOpened', 'Document', 'Files', 'View',
    'Clock', 'Setting', 'CircleCheck', 'Odometer', 'User', 'DocumentCopy',
    'ArrowLeft', 'ArrowRight', 'ArrowDown', 'ArrowUp', 'Close', 'Expand',
    'MoreFilled', 'Timer', 'Aim', 'Box', 'Grid', 'Tickets', 'Calendar',
    'Folder', 'VideoPlay', 'VideoPause', 'SuccessFilled', 'WarningFilled',
    'CircleCloseFilled', 'QuestionFilled', 'Coin', 'DataAnalysis', 'Filter',
    'FullScreen', 'House', 'Back', 'Bell', 'Bottom', 'Check', 'Loading',
    'WarnTriangleFilled', 'SwitchButton', 'Select', 'Shop', 'Guide', 'Picture',
    'Right', 'Link', 'Notebook', 'Memo', 'List', 'Finished', 'Sort', 'Rank',
    'Position', 'Pointer', 'Management', 'Operation', 'Stopwatch', 'DataBoard',
    'Histogram', 'TrendCharts', 'DataLine', 'Promotion', 'CopyDocument',
    'OfficeBuilding', 'PriceTag', 'CircleCheckFilled', 'Cpu', 'DocumentChecked',
    'ElLoading',
}

def clean_orphaned_imports(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        lines = f.readlines()

    new_lines = []
    i = 0
    changed = False

    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        # Check if this is the start of an orphaned import block
        if stripped == 'import {':
            # Look ahead to see if this is an icon import block
            j = i + 1
            is_icon_block = True
            block_end = None

            while j < len(lines):
                next_stripped = lines[j].strip()

                # Check if line is an icon name (with optional comma)
                name = next_stripped.rstrip(',').strip()
                if name in ICON_NAMES:
                    j += 1
                    continue
                elif next_stripped.startswith('}'):
                    # This has a proper closing - check if it's from element-plus
                    if 'element-plus' in next_stripped or 'icons-vue' in next_stripped:
                        block_end = j
                    else:
                        is_icon_block = False
                    break
                elif next_stripped.startswith('import ') or next_stripped == '':
                    # No closing brace found - this is an orphaned block
                    block_end = j - 1
                    # Find the last icon name line
                    while block_end >= i and lines[block_end].strip() == '':
                        block_end -= 1
                    break
                else:
                    is_icon_block = False
                    break

            if is_icon_block and block_end is not None:
                # Skip the entire orphaned block
                # If block_end points to a } from line, skip it too
                skip_to = block_end + 1
                if skip_to < len(lines) and lines[skip_to].strip().startswith('}'):
                    skip_to += 1
                i = skip_to
                changed = True
                continue

        # Check for standalone icon name lines (not in an import block)
        # These might be leftover from broken imports
        name = stripped.rstrip(',').strip()
        if name in ICON_NAMES and i > 0:
            prev = lines[i-1].strip() if i > 0 else ''
            prev_name = prev.rstrip(',').strip()
            if prev_name in ICON_NAMES or prev == 'import {':
                # This is part of an orphaned block, skip
                i += 1
                changed = True
                continue

        new_lines.append(line)
        i += 1

    if changed:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.writelines(new_lines)
        return True
    return False

# Process all files
vue_files = glob.glob(os.path.join(KUBE_DIR, '**/*.vue'), recursive=True)
cleaned = 0
for filepath in sorted(vue_files):
    if clean_orphaned_imports(filepath):
        print(f"Cleaned: {filepath}")
        cleaned += 1

print(f"\nCleaned {cleaned} files.")
