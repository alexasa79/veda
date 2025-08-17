#!/usr/bin/env bash
# Save as: show_ansi_palette.sh
# Usage: bash show_ansi_palette.sh

names=(black red green yellow blue magenta cyan white)

echo "TERM=$TERM"
echo

echo "Foreground (basic, 30–37):"
for i in {0..7}; do
  code=$((30+i))
  printf "\e[%sm%-7s\e[0m (%s)  " "$code" "${names[$i]}" "$code"
done
echo; echo

echo "Foreground (bright, 90–97):"
for i in {0..7}; do
  code=$((90+i))
  printf "\e[%sm%-7s\e[0m (%s)  " "$code" "bright-${names[$i]}" "$code"
done
echo; echo

echo "Background (basic, 40–47) with fg=white for readability:"
for i in {0..7}; do
  code=$((40+i))
  printf "\e[37;%sm %-10s \e[0m (%s)  " "$code" "${names[$i]}" "$code"
done
echo; echo

echo "Background (bright, 100–107) with fg=black for readability:"
for i in {0..7}; do
  code=$((100+i))
  printf "\e[30;%sm %-10s \e[0m (%s)  " "$code" "bright-${names[$i]}" "$code"
done
echo; echo

echo "Sample line mixing basic/bright foregrounds:"
printf "\e[31mred\e[0m "
printf "\e[34mblue\e[0m "
printf "\e[94mbright-blue\e[0m "
printf "\e[33myellow\e[0m "
printf "\e[93mbright-yellow\e[0m\n"
