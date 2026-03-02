#!/usr/bin/env python3
"""
Convert a-table-column (invalid in Arco) to proper a-table columns array format.
Handles the most common patterns found in the kubernetes Vue files.
"""
import re
import os
import glob

KUBE_DIR = "web/src/views/kubernetes"

def extract_table_columns(table_block):
    """Extract column definitions from a-table-column elements."""
    columns = []
    # Match a-table-column with various attributes
    col_pattern = re.compile(
        r'<a-table-column\s+(.*?)(?:/>|>)',
        re.DOTALL
    )

    for match in col_pattern.finditer(table_block):
        attrs_str = match.group(1).strip()
        col = {}

        # Extract label/title
        label_match = re.search(r'label="([^"]*)"', attrs_str)
        if label_match:
            col['title'] = label_match.group(1)

        # Extract prop/dataIndex
        prop_match = re.search(r'prop="([^"]*)"', attrs_str)
        if prop_match:
            col['dataIndex'] = prop_match.group(1)

        # Extract width
        width_match = re.search(r'(?:min-)?width="(\d+)"', attrs_str)
        if width_match:
            col['width'] = int(width_match.group(1))

        # Extract fixed
        fixed_match = re.search(r'fixed="([^"]*)"', attrs_str)
        if fixed_match:
            col['fixed'] = fixed_match.group(1)

        # Check for show-overflow-tooltip
        if 'show-overflow-tooltip' in attrs_str or 'ellipsis' in attrs_str:
            col['ellipsis'] = True
            col['tooltip'] = True

        # Extract align
        align_match = re.search(r'align="([^"]*)"', attrs_str)
        if align_match:
            col['align'] = align_match.group(1)

        # Check if it's a selection column
        type_match = re.search(r'type="selection"', attrs_str)
        if type_match:
            col['type'] = 'selection'

        columns.append(col)

    return columns

def has_custom_template(table_block, col_index):
    """Check if a column has a custom template."""
    # Find all a-table-column blocks
    col_blocks = re.findall(
        r'<a-table-column[^>]*>.*?</a-table-column>',
        table_block,
        re.DOTALL
    )
    if col_index < len(col_blocks):
        return '<template' in col_blocks[col_index]
    return False

def generate_slot_name(col):
    """Generate a slot name for a column."""
    if col.get('dataIndex'):
        return col['dataIndex']
    if col.get('title'):
        # Convert Chinese title to a simple slot name
        title = col['title']
        if title == '操作':
            return 'actions'
        if title == '状态':
            return 'status'
        return col.get('dataIndex', 'col_' + str(hash(title) % 1000))
    return None

def build_column_def(col, needs_slot=False, slot_name=None):
    """Build a column definition string."""
    parts = []

    if col.get('type') == 'selection':
        return None  # Selection is handled via row-selection prop

    if col.get('title'):
        parts.append(f"title: '{col['title']}'")

    if col.get('dataIndex'):
        parts.append(f"dataIndex: '{col['dataIndex']}'")

    if needs_slot and slot_name:
        parts.append(f"slotName: '{slot_name}'")

    if col.get('width'):
        parts.append(f"width: {col['width']}")

    if col.get('fixed'):
        parts.append(f"fixed: '{col['fixed']}'")

    if col.get('ellipsis'):
        parts.append("ellipsis: true")
        parts.append("tooltip: true")

    if col.get('align'):
        parts.append(f"align: '{col['align']}'")

    return '{ ' + ', '.join(parts) + ' }'

def convert_table_in_file(filepath):
    """Convert a-table-column to columns array in a Vue file."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    if 'a-table-column' not in content:
        return False

    # Find all table blocks
    # Pattern: <a-table ...> ... </a-table>
    table_pattern = re.compile(
        r'(<a-table\b[^>]*>)(.*?)(</a-table>)',
        re.DOTALL
    )

    tables_found = list(table_pattern.finditer(content))
    if not tables_found:
        return False

    # Process each table (in reverse order to preserve positions)
    new_content = content
    columns_arrays = []
    table_idx = 0

    for match in reversed(tables_found):
        table_open = match.group(1)
        table_body = match.group(2)
        table_close = match.group(3)

        if 'a-table-column' not in table_body:
            continue

        table_idx += 1
        col_suffix = '' if table_idx == 1 else str(table_idx)
        columns_var = f'tableColumns{col_suffix}'

        # Extract column blocks with their content
        col_blocks = []
        col_pattern = re.compile(
            r'<a-table-column\s+(.*?)(?:/>|>(.*?)</a-table-column>)',
            re.DOTALL
        )

        for col_match in col_pattern.finditer(table_body):
            attrs_str = col_match.group(1).strip()
            template_content = col_match.group(2) or ''

            col = {}

            # Extract attributes
            label_m = re.search(r'label="([^"]*)"', attrs_str)
            if label_m: col['title'] = label_m.group(1)

            prop_m = re.search(r'prop="([^"]*)"', attrs_str)
            if prop_m: col['dataIndex'] = prop_m.group(1)

            width_m = re.search(r'width="(\d+)"', attrs_str)
            if width_m: col['width'] = int(width_m.group(1))

            minwidth_m = re.search(r'min-width="(\d+)"', attrs_str)
            if minwidth_m: col['width'] = int(minwidth_m.group(1))

            fixed_m = re.search(r'fixed="([^"]*)"', attrs_str)
            if fixed_m: col['fixed'] = fixed_m.group(1)

            align_m = re.search(r'align="([^"]*)"', attrs_str)
            if align_m: col['align'] = align_m.group(1)

            if 'show-overflow-tooltip' in attrs_str:
                col['ellipsis'] = True
                col['tooltip'] = True

            if 'type="selection"' in attrs_str:
                col['is_selection'] = True

            has_template = '<template' in template_content
            col['has_template'] = has_template
            col['template_content'] = template_content.strip()

            # Generate slot name
            if has_template:
                if col.get('dataIndex'):
                    col['slotName'] = col['dataIndex']
                elif col.get('title') == '操作':
                    col['slotName'] = 'actions'
                elif col.get('title'):
                    # Create a simple slug
                    slug = col['title'].replace(' ', '_')
                    col['slotName'] = f'col_{slug}'
                else:
                    col['slotName'] = f'col_{len(col_blocks)}'

            col_blocks.append(col)

        # Build columns array
        col_defs = []
        has_selection = False
        for col in col_blocks:
            if col.get('is_selection'):
                has_selection = True
                continue

            parts = []
            if col.get('title'):
                parts.append(f"title: '{col['title']}'")
            if col.get('dataIndex'):
                parts.append(f"dataIndex: '{col['dataIndex']}'")
            if col.get('has_template') and col.get('slotName'):
                parts.append(f"slotName: '{col['slotName']}'")
            if col.get('width'):
                parts.append(f"width: {col['width']}")
            if col.get('fixed'):
                parts.append(f"fixed: '{col['fixed']}'")
            if col.get('ellipsis'):
                parts.append("ellipsis: true, tooltip: true")
            if col.get('align'):
                parts.append(f"align: '{col['align']}'")

            col_defs.append('  { ' + ', '.join(parts) + ' }')

        columns_def = f"const {columns_var} = [\n" + ',\n'.join(col_defs) + "\n]"
        columns_arrays.append(columns_def)

        # Build new table template
        # Add :columns prop to table opening tag
        new_table_open = table_open.rstrip('>')
        # Remove v-loading if present (already handled)
        new_table_open += f' :columns="{columns_var}"'
        if has_selection:
            if ':row-selection' not in new_table_open:
                new_table_open += ' :row-selection="{ type: \'checkbox\', showCheckedAll: true }"'
        new_table_open += '>'

        # Build slot templates from columns with custom templates
        slot_templates = []
        for col in col_blocks:
            if col.get('is_selection'):
                continue
            if col.get('has_template') and col.get('slotName'):
                tmpl = col['template_content']
                # Convert template #default="{ row }" to #slotName="{ record }"
                # and replace row. with record.
                tmpl = re.sub(
                    r'<template\s+#default="{\s*row\s*(?:,\s*\$index)?\s*}">',
                    f'<template #{col["slotName"]}="{{{{ record }}}}">',
                    tmpl
                )
                tmpl = re.sub(
                    r'<template\s+#default="{\s*row,\s*\$index\s*}">',
                    f'<template #{col["slotName"]}="{{{{ record, rowIndex }}}}">',
                    tmpl
                )
                tmpl = re.sub(
                    r'<template\s+#default="{\s*\$index\s*}">',
                    f'<template #{col["slotName"]}="{{{{ rowIndex }}}}">',
                    tmpl
                )
                # Replace row. references with record.
                tmpl = tmpl.replace('row.', 'record.')
                tmpl = tmpl.replace('(row)', '(record)')
                tmpl = tmpl.replace('(row,', '(record,')
                tmpl = tmpl.replace(' row"', ' record"')
                tmpl = tmpl.replace('$index', 'rowIndex')

                # Also handle #header templates
                tmpl = re.sub(r'<template\s+#header>', '<template #header>', tmpl)

                slot_templates.append(tmpl)

        new_table_body = '\n          '.join(slot_templates) if slot_templates else ''

        # Replace the table block
        new_table = new_table_open + '\n          ' + new_table_body + '\n        ' + table_close
        new_content = new_content[:match.start()] + new_table + new_content[match.end():]

    # Add columns arrays to script section
    if columns_arrays:
        # Find the script setup section
        script_match = re.search(r'(<script\s+setup[^>]*>)', new_content)
        if script_match:
            insert_pos = script_match.end()
            columns_code = '\n' + '\n\n'.join(reversed(columns_arrays)) + '\n'
            new_content = new_content[:insert_pos] + columns_code + new_content[insert_pos:]

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(new_content)

    return True

# Process all files
vue_files = glob.glob(os.path.join(KUBE_DIR, '**/*.vue'), recursive=True)
converted = 0
for filepath in sorted(vue_files):
    if convert_table_in_file(filepath):
        print(f"Converted tables in: {filepath}")
        converted += 1

print(f"\nDone! Converted tables in {converted} files.")
