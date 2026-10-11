# Demonstrates the order saga end-to-end through the nginx gateway.
# Prerequisite: docker compose up --build
#
#   powershell -ExecutionPolicy Bypass -File scripts\saga-demo.ps1
#
# Scenario 1: cheap order   -> PaymentCompleted -> order PAID
# Scenario 2: costly order  -> PaymentFailed    -> compensation: order CANCELLED
# Scenario 3: cancel a PAID order               -> PaymentRefunded

param([string]$BaseUrl = "http://localhost:8080")

$ErrorActionPreference = "Stop"

function Post($path, $body) {
    Invoke-RestMethod -Method Post -Uri "$BaseUrl$path" -ContentType "application/json" -Body ($body | ConvertTo-Json -Depth 5)
}

function Wait-OrderStatus($orderId, [string[]]$expected) {
    for ($i = 0; $i -lt 30; $i++) {
        $o = Invoke-RestMethod "$BaseUrl/api/orders/$orderId"
        if ($expected -contains $o.status) { return $o }
        Start-Sleep -Milliseconds 500
    }
    throw "order $orderId did not reach $($expected -join '/') (last: $($o.status))"
}

$userId = [guid]::NewGuid().ToString()
Write-Host "user: $userId`n"

# Products: one under and one over the payment limit (PAYMENT_MAX_AMOUNT=1000.00).
$cheap  = Post "/api/catalog/products" @{ name = "Mouse";  description = "demo"; price = 1500;   stock = 10 }
$costly = Post "/api/catalog/products" @{ name = "Laptop"; description = "demo"; price = 250000; stock = 10 }

Write-Host "== Scenario 1: payment succeeds"
$o1 = Post "/api/orders" @{ userId = $userId; items = @(@{ productId = $cheap.id; quantity = 2 }) }
Write-Host "created $($o1.id): $($o1.status), total $($o1.total / 100)"
$o1 = Wait-OrderStatus $o1.id @("PAID", "CANCELLED")
Write-Host "saga result: $($o1.status) (payment $($o1.paymentId))`n"

Write-Host "== Scenario 2: payment fails -> compensation"
$o2 = Post "/api/orders" @{ userId = $userId; items = @(@{ productId = $costly.id; quantity = 1 }) }
Write-Host "created $($o2.id): $($o2.status), total $($o2.total / 100)"
$o2 = Wait-OrderStatus $o2.id @("PAID", "CANCELLED")
Write-Host "saga result: $($o2.status) - $($o2.cancelReason)`n"

Write-Host "== Scenario 3: cancel a paid order -> refund"
Invoke-RestMethod -Method Patch -Uri "$BaseUrl/api/orders/$($o1.id)/status" -ContentType "application/json" -Body '{"status":"CANCELLED"}' | Out-Null
Start-Sleep -Seconds 2
$p = Invoke-RestMethod "$BaseUrl/api/payments?orderId=$($o1.id)"
Write-Host "payment of $($o1.id): $($p.status)`n"

Write-Host "== Notifications sent to the user"
Start-Sleep -Seconds 1
(Invoke-RestMethod "$BaseUrl/api/notifications?userId=$userId").items |
    Sort-Object createdAt |
    ForEach-Object { Write-Host "  [$($_.event)] $($_.message)" }
