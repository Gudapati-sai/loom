# make-screenshot.ps1 — renders a captured terminal session (ANSI) into a
# professional PNG screenshot with a terminal window chrome. Used to
# regenerate the images in /screenshots from /captures.
#
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts/make-screenshot.ps1 `
#     -InFile captures/01-menu.txt -OutFile screenshots/01-menu.png -Title "loom — menu"

param(
  [Parameter(Mandatory = $true)][string]$InFile,
  [Parameter(Mandatory = $true)][string]$OutFile,
  [string]$Title = "loom"
)

Add-Type -AssemblyName System.Drawing

$ESC = [char]27

# --- palette (GitHub-dark terminal + Claude terracotta accent) ---
$colAccent = [System.Drawing.Color]::FromArgb(255, 217, 151, 87)  # #D97757 terracotta
$colGreen  = [System.Drawing.Color]::FromArgb(255, 63, 185, 80)   # #3FB950
$colYellow = [System.Drawing.Color]::FromArgb(255, 210, 153, 34)  # #D29922
$colCyan   = [System.Drawing.Color]::FromArgb(255, 88, 166, 255)  # #58A6FF
$colGray   = [System.Drawing.Color]::FromArgb(255, 139, 148, 158) # #8B949E
$colFg     = [System.Drawing.Color]::FromArgb(255, 230, 237, 243) # #E6EDF3
$colBg     = [System.Drawing.Color]::FromArgb(255, 13, 17, 23)    # #0D1117
$colBar    = [System.Drawing.Color]::FromArgb(255, 33, 38, 45)    # #21262D

function BasicColor([int]$n) {
  $map = @(
    (0, 0, 0), (205, 49, 49), (13, 188, 121), (229, 229, 16),
    (36, 114, 200), (188, 63, 188), (17, 168, 205), (229, 229, 229),
    (102, 102, 102), (241, 76, 76), (35, 209, 139), (245, 245, 67),
    (59, 142, 234), (214, 112, 214), (41, 184, 219), (255, 255, 255)
  )
  if ($n -lt $map.Count) { $c = $map[$n]; return [System.Drawing.Color]::FromArgb(255, $c[0], $c[1], $c[2]) }
  return $colFg
}

function XtermColor([int]$n) {
  if ($n -lt 16) { return (BasicColor $n) }
  if ($n -ge 232) { $v = 8 + ($n - 232) * 10; return [System.Drawing.Color]::FromArgb(255, $v, $v, $v) }
  $n = $n - 16
  $r = [math]::Floor($n / 36); $g = [math]::Floor(($n % 36) / 6); $b = $n % 6
  $cv = { param($x) if ($x -eq 0) { 0 } else { 55 + $x * 40 } }
  $rv = & $cv $r; $gv = & $cv $g; $bv = & $cv $b
  return [System.Drawing.Color]::FromArgb(255, $rv, $gv, $bv)
}

# Apply an SGR parameter string to the current color.
function Apply-SGR([string]$params, $cur) {
  $parts = $params -split ';'
  $new = $cur
  $p = 0
  while ($p -lt $parts.Count) {
    $n = 0
    [void][int]::TryParse($parts[$p], [ref]$n)
    switch ($n) {
      0  { $new = $colFg }
      31 { $new = $colAccent }  # red / bright red -> Claude terracotta accent
      91 { $new = $colAccent }
      32 { $new = $colGreen }
      92 { $new = $colGreen }
      33 { $new = $colYellow }
      93 { $new = $colYellow }
      36 { $new = $colCyan }
      96 { $new = $colCyan }
      37 { $new = $colFg }
      97 { $new = $colFg }
      90 { $new = $colGray }
      38 {
        if ($p + 1 -lt $parts.Count -and $parts[$p + 1] -eq '2' -and $p + 4 -lt $parts.Count) {
          $new = [System.Drawing.Color]::FromArgb(255, [int]$parts[$p + 2], [int]$parts[$p + 3], [int]$parts[$p + 4])
          $p += 4
        } elseif ($p + 1 -lt $parts.Count -and $parts[$p + 1] -eq '5' -and $p + 2 -lt $parts.Count) {
          $new = XtermColor ([int]$parts[$p + 2])
          $p += 2
        }
      }
      39 { $new = $colFg }
    }
    $p++
  }
  return $new
}

# Parse one line into (text, color) segments, dropping cursor/mode sequences.
function Parse-Line([string]$line) {
  $segs = New-Object System.Collections.ArrayList
  $cur = $colFg
  $text = New-Object System.Text.StringBuilder
  $i = 0
  while ($i -lt $line.Length) {
    $c = $line[$i]
    if ($c -eq $ESC -and $i + 1 -lt $line.Length -and $line[$i + 1] -eq '[') {
      $j = $i + 2
      while ($j -lt $line.Length -and $line[$j] -notmatch '[A-Za-z]') { $j++ }
      if ($j -lt $line.Length) {
        $seq = $line.Substring($i + 2, $j - $i - 2)
        if ($line[$j] -eq 'm') {
          if ($text.Length -gt 0) { [void]$segs.Add(@($text.ToString(), $cur)); [void]$text.Clear() }
          $cur = Apply-SGR $seq $cur
        }
        $i = $j + 1
        continue
      }
    }
    [void]$text.Append($c)
    $i++
  }
  if ($text.Length -gt 0) { [void]$segs.Add(@($text.ToString(), $cur)) }
  return ,$segs
}

# --- load and clean the capture ---
$raw = [System.IO.File]::ReadAllText($InFile)
$lines = @($raw -split "`n" | ForEach-Object { $_.TrimEnd("`r") })
$lines = @($lines | Where-Object { $_ -notmatch '^Assertion failed' })
while ($lines.Count -gt 0 -and $lines[$lines.Count - 1].Trim() -eq '') {
  $lines = $lines[0..($lines.Count - 2)]
}

# --- measure ---
$font     = New-Object System.Drawing.Font('Consolas', 15)
$fontBold = New-Object System.Drawing.Font('Consolas', 15, [System.Drawing.FontStyle]::Bold)
$measure  = New-Object System.Drawing.Bitmap(1, 1)
$mg       = [System.Drawing.Graphics]::FromImage($measure)
$sf       = New-Object System.Drawing.StringFormat([System.Drawing.StringFormat]::GenericTypographic)
$charW    = $mg.MeasureString('M', $font, 1000, $sf).Width
$lineH    = [math]::Ceiling($mg.MeasureString('Mg', $font, 1000, $sf).Height)

$maxLen = 0
foreach ($l in $lines) {
  $len = 0
  foreach ($s in (Parse-Line $l)) { $len += $s[0].Length }
  if ($len -gt $maxLen) { $maxLen = $len }
}

$padX = 18; $padY = 14; $barH = 38
$W = [int]($maxLen * $charW + 2 * $padX)
$H = [int]($barH + $lines.Count * $lineH + 2 * $padY)

$bmp = New-Object System.Drawing.Bitmap($W, $H)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.Clear($colBg)
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit

# --- terminal window chrome ---
$barBrush = New-Object System.Drawing.SolidBrush($colBar)
$g.FillRectangle($barBrush, 0, 0, $W, $barH)
$barBrush.Dispose()
$dotCols = @(
  (255, 95, 95),    # close (red)
  (255, 189, 77),   # minimize (yellow)
  (80, 200, 120)    # maximize (green)
)
$dotY = ($barH / 2) - 5
$dotX = 18
foreach ($dc in $dotCols) {
  $b = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, $dc[0], $dc[1], $dc[2]))
  $g.FillEllipse($b, $dotX, $dotY, 10, 10)
  $b.Dispose()
  $dotX += 18
}
$titleBrush = New-Object System.Drawing.SolidBrush($colGray)
$tw = $mg.MeasureString($Title, $fontBold, 1000, $sf).Width
$g.DrawString($Title, $fontBold, $titleBrush, ($W - $tw) / 2, ($barH - $lineH) / 2, $sf)
$titleBrush.Dispose()

# --- content ---
$y = $barH + $padY
foreach ($l in $lines) {
  $x = $padX
  foreach ($seg in (Parse-Line $l)) {
    $brush = New-Object System.Drawing.SolidBrush($seg[1])
    $g.DrawString($seg[0], $font, $brush, $x, $y, $sf)
    $x += $mg.MeasureString($seg[0], $font, 1000, $sf).Width
    $brush.Dispose()
  }
  $y += $lineH
}

$dir = Split-Path -Parent $OutFile
if ($dir -and -not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
$bmp.Save($OutFile, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose(); $mg.Dispose(); $bmp.Dispose(); $measure.Dispose()
Write-Host "wrote $OutFile (${W}x${H})"
