param(
    [string]$BaseUrl = "http://localhost:8080/api/v1"
)

function CallJson {
    param(
        [ValidateSet('GET','POST','PATCH','DELETE')][string]$Method,
        [string]$Path,
        $Body
    )
    $uri = "$BaseUrl$Path"
    if ($null -ne $Body) {
        $json = if ($Body -is [string]) { $Body } else { $Body | ConvertTo-Json -Depth 6 }
        return Invoke-RestMethod -Method $Method -Uri $uri -Body $json -ContentType "application/json" -ErrorAction Stop
    } else {
        return Invoke-RestMethod -Method $Method -Uri $uri -ErrorAction Stop
    }
}

try {
    Write-Host "[demo-flow] Health: $BaseUrl/health"
    CallJson GET "/health" $null | Out-Null

    # Optional: login as admin to demonstrate auth works (not required for flow)
    try {
        $loginRes = CallJson POST "/auth/login" @{ name = "admin"; password = "admin" }
        if ($loginRes -and $loginRes.access_token) { Write-Host "[demo-flow] Login success (admin)" }
    } catch { Write-Host "[demo-flow] Login skipped: $($_.Exception.Message)" }

    # 1) Create order
    $now = (Get-Date).ToUniversalTime().ToString("s") + "Z"  # RFC3339
    $orderNumber = "ORD-" + [DateTime]::UtcNow.Ticks
    $orderBody = @{ 
        order_number = $orderNumber
        style_number = "STYLE-PS1"
        customer_name = "ACME"
        order_start_date = $now
        note = "order via script"
        items = @(
            @{ color = "Red";  size = "M"; quantity = 10 },
            @{ color = "Blue"; size = "L"; quantity = 5 }
        )
    }
    $order = CallJson POST "/orders" $orderBody
    Write-Host "[demo-flow] Order created: id=$($order.order_id) number=$($order.order_number)"

    # 2) Create plan
    $plan = CallJson POST "/plans" @{ plan_name = "Plan-A"; order_id = $order.order_id }
    Write-Host "[demo-flow] Plan created: id=$($plan.plan_id) status=$($plan.status)"

    # 3) Create layout
    $layout = CallJson POST "/layouts" @{ layout_name = "Layout-1"; plan_id = $plan.plan_id }
    Write-Host "[demo-flow] Layout created: id=$($layout.layout_id)"

    # 4) Create tasks (colors must exist in order items)
    $taskRed  = CallJson POST "/tasks" @{ layout_id = $layout.layout_id; color = "Red";  planned_layers = 10 }
    $taskBlue = CallJson POST "/tasks" @{ layout_id = $layout.layout_id; color = "Blue"; planned_layers = 5 }
    Write-Host "[demo-flow] Tasks created: red=$($taskRed.task_id) blue=$($taskBlue.task_id)"

    # 5) Publish plan
    CallJson POST "/plans/$($plan.plan_id)/publish" $null | Out-Null
    Write-Host "[demo-flow] Plan published"

    # 6) Create logs (use worker_name to avoid user ID dependency)
    $logRed  = CallJson POST "/logs" @{ task_id = $taskRed.task_id;  layers_completed = 9; worker_name = "Zhang San"; note = "done red" }
    $logBlue = CallJson POST "/logs" @{ task_id = $taskBlue.task_id; layers_completed = 4;  worker_name = "Wang Wu"; note = "done blue" }
    Write-Host "[demo-flow] Logs created: red=$($logRed.log_id) blue=$($logBlue.log_id)"

    # 7) Participants list (aggregated from current task logs)
    $participants = CallJson GET "/tasks/$($taskRed.task_id)/participants" $null
    Write-Host "[demo-flow] Participants(red): $($participants | ConvertTo-Json -Compress)"

    # 8) Void red log (voided_by optional; fall back if not found)
    $adminId = $null
    try {
        $usersAdmin = CallJson GET "/users?name=admin" $null
        if ($usersAdmin -is [System.Array]) {
            if ($usersAdmin.Count -gt 0) { $adminId = $usersAdmin[0].user_id }
        } elseif ($usersAdmin -and $usersAdmin.user_id) {
            $adminId = $usersAdmin.user_id
        }
    } catch { $adminId = $null }
    if (-not $adminId) {
        try {
            $usersMgr = CallJson GET "/users?name=manager" $null
            if ($usersMgr -is [System.Array]) {
                if ($usersMgr.Count -gt 0) { $adminId = $usersMgr[0].user_id }
            } elseif ($usersMgr -and $usersMgr.user_id) {
                $adminId = $usersMgr.user_id
            }
        } catch { $adminId = $null }
    }
    $voidBody = @{ void_reason = "wrong color count" }
    if ($adminId) { $voidBody.voided_by = [int]$adminId }
    CallJson PATCH "/logs/$($logRed.log_id)" $voidBody | Out-Null
    Write-Host "[demo-flow] Red log voided (voided_by=$adminId)"

    # 9) Check task status and completed layers to confirm triggers
    $taskRedAfter = CallJson GET "/tasks/$($taskRed.task_id)" $null
    Write-Host "[demo-flow] Red task: status=$($taskRedAfter.status) completed_layers=$($taskRedAfter.completed_layers)"

    # 10) Freeze plan (further changes restricted, just demo)
    try {
        CallJson POST "/plans/$($plan.plan_id)/freeze" $null | Out-Null
        Write-Host "[demo-flow] Plan frozen"
    } catch {
        Write-Host "[demo-flow] Freeze skipped: $($_.Exception.Message)"
    }

    Write-Host "[demo-flow] Flow completed"
    Write-Host "Order=$($order.order_id) Plan=$($plan.plan_id) Layout=$($layout.layout_id) TaskRed=$($taskRed.task_id) LogRed=$($logRed.log_id)"
} catch {
    Write-Error "[demo-flow] Error: $($_.Exception.Message)"
    throw
}