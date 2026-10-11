<#
  Crea invitados de prueba en la app de Halloween y muestra sus contraseñas.

  Uso (desde la carpeta halloween-back-end, con el backend levantado):
    powershell -ExecutionPolicy Bypass -File scripts\crear-invitados.ps1

  Opciones:
    -AdminPassword maga42   Contraseña de un admin. Hace falta si ya hay invitados
                            cargados; si no la pasás, el script te la pide.
    -Votos 3                Votos para cada invitado nuevo (por defecto 2).
    -Api http://...         Dirección del backend (por defecto http://localhost:8080/api).

  Si la base está vacía, primero crea a Maga como admin.
  Para cambiar la lista, editá $invitados más abajo.
#>
param(
  [string]$Api = "http://localhost:8080/api",
  [string]$AdminPassword = "",
  [int]$Votos = 2
)

$ErrorActionPreference = "Stop"

$invitados = @(
  @{ name = "Lu";         costume = "Bruja del bosque" },
  @{ name = "Tomi";       costume = "Zombie oficinista" },
  @{ name = "Sai";        costume = "Payaso" },
  @{ name = "Dei";        costume = "Vampira" },
  @{ name = "María José"; costume = "Momia con estilo" },
  @{ name = "Nico";       costume = "Fantasma de la ópera" },
  @{ name = "Chiara";     costume = "Coraline" },
  @{ name = "Fede";       costume = "Hombre lobo" },
  @{ name = "Juli";       costume = "Calavera mexicana" },
  @{ name = "Pato";       costume = "Espantapájaros" }
)

function Send-Json($method, $url, $body, $password) {
  $headers = @{}
  if ($password) { $headers["User"] = $password }
  $json = ConvertTo-Json -InputObject @($body) -Depth 5
  # Se manda en UTF-8 para que los acentos lleguen bien.
  $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
  Invoke-RestMethod -Method $method -Uri $url -Headers $headers -Body $bytes -ContentType "application/json; charset=utf-8"
}

function Get-StatusCode($err) {
  try { return [int]$err.Exception.Response.StatusCode } catch { return 0 }
}

# ¿Ya hay invitados cargados?
try {
  # ForEach-Object desarma la lista, así una lista vacía cuenta como 0.
  $existentes = @(Invoke-RestMethod -Uri "$Api/users" | ForEach-Object { $_ })
} catch {
  Write-Host "No pude conectarme al backend en $Api. ¿Está levantado (docker compose up)?" -ForegroundColor Red
  exit 1
}

$nuevos = @($invitados | ForEach-Object {
  @{ name = $_.name; costume = $_.costume; pending_votes = $Votos; is_admin = $false }
})

if ($existentes.Count -eq 0) {
  Write-Host "La base está vacía: creo a Maga como admin." -ForegroundColor Yellow
  $nuevos = @(@{ name = "Maga"; costume = "Calabaza encantada"; pending_votes = $Votos; is_admin = $true }) + $nuevos
} else {
  # No repetir a quienes ya están (mismo nombre).
  $nombres = @($existentes | ForEach-Object { $_.name })
  $nuevos = @($nuevos | Where-Object { $nombres -notcontains $_.name })
  if ($nuevos.Count -eq 0) {
    Write-Host "Todos los invitados de la lista ya existen. No hay nada para crear." -ForegroundColor Yellow
    exit 0
  }
  if (-not $AdminPassword) {
    $AdminPassword = Read-Host "Ya hay invitados cargados. Contraseña de un admin (ej: la de Maga)"
  }
}

try {
  $respuesta = Send-Json "Post" "$Api/users/batch" $nuevos $AdminPassword
} catch {
  $code = Get-StatusCode $_
  if ($code -eq 401 -or $code -eq 403) {
    Write-Host "Esa contraseña no es de un admin. Probá de nuevo con -AdminPassword." -ForegroundColor Red
  } else {
    Write-Host "No se pudieron crear los invitados: $($_.Exception.Message)" -ForegroundColor Red
  }
  exit 1
}

$creados = @($respuesta.Users)
Write-Host ""
Write-Host "Creé $($creados.Count) invitados:" -ForegroundColor Green
Write-Host ("  {0,-16} {1,-26} {2}" -f "Nombre", "Disfraz", "Contraseña") -ForegroundColor DarkGray
foreach ($u in $creados) {
  $admin = ""
  if ($u.is_admin) { $admin = "  (admin)" }
  Write-Host ("  {0,-16} {1,-26} {2}{3}" -f $u.name, $u.costume, $u.pw_code, $admin)
}
Write-Host ""

$texto = ($creados | ForEach-Object { "$($_.name): $($_.pw_code)" }) -join "`r`n"
try {
  Set-Clipboard -Value $texto
  Write-Host "Copié la lista de contraseñas al portapapeles." -ForegroundColor Green
} catch {}
