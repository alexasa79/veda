#!/usr/bin/env bash
# Save as: show_ansi_palette_raw.sh
# Usage: bash show_ansi_palette_raw.sh

set -euo pipefail

names=(black red green yellow blue magenta cyan white)

hex_of() {
  # Print hex bytes of stdin using od (portable)
  od -An -v -tx1 | tr -d '\n' | sed -e 's/^ *//' -e 's/  */ /g'
}

echo "TERM=$TERM"
echo

echo "Foreground (basic, 30–37):"
for i in {0..7}; do
  code=$((30+i))
  seq="$(printf '\e[%sm' "$code")"
  reset="$(printf '\e[0m')"
  printf "%-14s " "${names[$i]}($code)"
  printf "%b%s%b  " "$seq" "${names[$i]}" "$reset"
  printf "[hex: "; printf "%s" "$seq" | hex_of; printf "]\n"
done

echo

echo "Foreground (bright, 90–97):"
for i in {0..7}; do
  code=$((90+i))
  seq="$(printf '\e[%sm' "$code")"
  reset="$(printf '\e[0m')"
  printf "%-20s " "bright-${names[$i]}($code)"
  printf "%b%s%b  " "$seq" "bright-${names[$i]}" "$reset"
  printf "[hex: "; printf "%s" "$seq" | hex_of; printf "]\n"
done

echo

echo "Background (basic, 40–47) with fg=white:"
for i in {0..7}; do
  code=$((40+i))
  seq="$(printf '\e[%sm' "$code")"
  fg="$(printf '\e[37m')"
  reset="$(printf '\e[0m')"
  printf "%-14s " "${names[$i]}($code)"
  printf "%b%b %-10s %b  " "$fg" "$seq" "${names[$i]}" "$reset"
  printf "[hex: "; printf "%s" "$seq" | hex_of; printf "]\n"
done

echo

echo "Background (bright, 100–107) with fg=black:"
for i in {0..7}; do
  code=$((100+i))
  seq="$(printf '\e[%sm' "$code")"
  fg="$(printf '\e[30m')"
  reset="$(printf '\e[0m')"
  printf "%-20s " "bright-${names[$i]}($code)"
  printf "%b%b %-12s %b  " "$fg" "$seq" "bright-${names[$i]}" "$reset"
  printf "[hex: "; printf "%s" "$seq" | hex_of; printf "]\n"
done
