# Integration Test and Verification Guide

This document provides a comprehensive guide for testing the notification system integration between the Go backend and Next.js frontend.

## Prerequisites

Before starting the tests, ensure you have:

1. PostgreSQL database running (via Docker Compose)
2. Go backend dependencies installed
3. Next.js frontend dependencies installed (`npm install` in `examples/notification/frontend`)

## Test Environment Setup

### 1. Start PostgreSQL Database

```bash
cd examples/notification
docker compose up -d postgres
```

Verify the database is running:
```bash
docker compose ps
```

### 2. Start Go Backend (Terminal 1)

```bash
cd examples/notification
./run.sh
```

The backend should start on `http://localhost:8080`

Expected output:
```
Starting notification service...
Server listening on :8080
```

### 3. Start Next.js Frontend (Terminal 2)

```bash
cd examples/notification/frontend
npm run dev
```

The frontend should start on `http://localhost:3000`

Expected output:
```
▲ Next.js 15.x.x
- Local:        http://localhost:3000
```

## Integration Test Scenarios

### Test 1: Success Job Execution and Notification Display

**Objective:** Verify that a success job creates a notification that appears in real-time.

**Steps:**
1. Open browser to `http://localhost:3000`
2. Check the notification icon in the header (bell icon)
3. Initial state: Badge should show 0 or no badge
4. Click the "非同期ジョブを実行（成功）" button
5. Wait 3 seconds for the job to complete

**Expected Results:**
- ✅ Button shows "実行中..." during execution
- ✅ Button becomes enabled after completion
- ✅ Notification badge appears with count "1" (or increments)
- ✅ Badge appears within 1 second of job completion (SSE real-time update)
- ✅ Click notification icon to open dropdown
- ✅ Dropdown shows notification with title "ジョブ完了"
- ✅ Notification body: "ジョブが正常に完了しました"
- ✅ Notification is NOT marked as important (no red indicator)
- ✅ Notification background is highlighted (unread state)

**Verification Commands:**
```bash
# Check backend logs for notification creation
# Should see: POST /api/notifications - 201

# Check SSE connection in browser DevTools Network tab
# Should see: notifications/users/EMP001/stream (EventStream)
```

---

### Test 2: Error Job Execution and Important Notification

**Objective:** Verify that an error job creates an important notification.

**Steps:**
1. From the same page, click "非同期ジョブを実行（エラー）" button
2. Wait 3 seconds for the job to complete

**Expected Results:**
- ✅ Button shows "実行中..." during execution
- ✅ Notification badge increments (e.g., from 1 to 2)
- ✅ New notification appears in dropdown automatically
- ✅ Notification title: "ジョブエラー"
- ✅ Notification body: "ジョブの実行中にエラーが発生しました"
- ✅ Notification has RED indicator (important flag)
- ✅ Notification is unread (highlighted background)

**Verification:**
- Open browser console and check for SSE event:
  ```json
  {
    "type": "notification",
    "payload": {
      "id": 2,
      "title": "ジョブエラー",
      "important": true,
      ...
    }
  }
  ```

---

### Test 3: Fix Job - Notification Update and Read Status

**Objective:** Verify that the fix job updates the error notification and marks it as read.

**Steps:**
1. Click "最後のエラージョブを修正" button
2. Wait 3 seconds for the job to complete
3. Observe the notification dropdown

**Expected Results:**
- ✅ Button shows "実行中..." during execution
- ✅ The error notification title changes to "ジョブエラー（解決済み）"
- ✅ The notification body changes to "問題は解決されました"
- ✅ The RED indicator disappears (important: false)
- ✅ The notification is marked as read (no highlighted background)
- ✅ Badge count decrements (important notification was marked as read)
- ✅ Update happens in real-time via SSE

**Verification:**
- Check browser console for two SSE events:
  1. Update event: `{"type":"update","payload":{...}}`
  2. Read event: `{"type":"read","payload":{...}}`

---

### Test 4: SSE Connection and Real-Time Updates

**Objective:** Verify SSE connection is working correctly.

**Steps:**
1. Open browser DevTools → Network tab
2. Filter by "stream" or look for EventStream type
3. Refresh the page
4. Look for `GET /api/notifications/users/EMP001/stream`

**Expected Results:**
- ✅ SSE connection is established (Status: 200, Type: eventsource)
- ✅ Connection shows "Pending" (stays open)
- ✅ Keep-alive comments are sent every 30 seconds (`: keep-alive`)
- ✅ When a job is executed, events appear immediately
- ✅ If connection is lost, automatic reconnection occurs within 5 seconds

**Manual Reconnection Test:**
1. Stop the Go backend (Ctrl+C in Terminal 1)
2. Observe browser console: Should see reconnection attempts
3. Restart the Go backend (`./run.sh`)
4. Connection should re-establish automatically
5. Execute a job to verify events are received

---

### Test 5: Read Status Management

**Objective:** Verify that read status is properly managed.

**Steps:**
1. Execute a success job to create a non-important notification
2. Click the notification icon to open dropdown
3. Observe the notification list

**Expected Results:**
- ✅ Non-important unread notifications are automatically marked as read when dropdown opens
- ✅ Badge count decrements for non-important notifications
- ✅ Notification background changes from highlighted to normal
- ✅ API call is made: `POST /api/users/EMP001/notifications/mark-non-important`

**Important Notification Read Test:**
1. Execute an error job (creates important notification)
2. Open dropdown (important notification should remain unread)
3. Click on the important notification to open detail modal
4. Click "既読にする" button

**Expected Results:**
- ✅ Modal shows "既読にする" button for important unread notifications
- ✅ After clicking, notification is marked as read
- ✅ Badge count decrements
- ✅ Modal closes or button disappears
- ✅ API call: `POST /api/users/EMP001/notifications/{id}/read`

---

### Test 6: Notification Detail Modal

**Objective:** Verify notification detail modal functionality.

**Steps:**
1. Create multiple notifications (success and error jobs)
2. Open notification dropdown
3. Click on a notification

**Expected Results:**
- ✅ Modal opens with full notification details
- ✅ Title is displayed correctly
- ✅ Body text is displayed correctly
- ✅ Created date/time is shown
- ✅ Important indicator is shown for important notifications
- ✅ For important unread notifications, "既読にする" button is visible
- ✅ Click outside modal or close button to dismiss
- ✅ Modal closes properly

---

### Test 7: Responsive Design

**Objective:** Verify responsive design works on different screen sizes.

**Steps:**
1. Open browser DevTools → Toggle device toolbar (Cmd+Shift+M / Ctrl+Shift+M)
2. Test different screen sizes:
   - Mobile: 375px width (iPhone SE)
   - Tablet: 768px width (iPad)
   - Desktop: 1920px width

**Expected Results:**

**Mobile (< 640px):**
- ✅ Notification dropdown is full width
- ✅ Notification icon is properly sized
- ✅ Modal is full width with proper padding
- ✅ Job trigger buttons stack vertically
- ✅ Text is readable without horizontal scroll

**Tablet (640px - 1024px):**
- ✅ Notification dropdown has fixed width (360px)
- ✅ Layout is properly centered
- ✅ All interactive elements are easily tappable

**Desktop (> 1024px):**
- ✅ Notification dropdown positioned correctly (right-aligned)
- ✅ Modal is centered with max-width
- ✅ Hover states work on buttons and notifications

---

### Test 8: Error Handling

**Objective:** Verify error handling and user feedback.

**Test 8.1: Backend Unavailable**
1. Stop the Go backend
2. Try to execute a job
3. Observe error messages

**Expected Results:**
- ✅ Error message appears: "ネットワークエラーが発生しました" or similar
- ✅ Button returns to enabled state
- ✅ User can retry

**Test 8.2: SSE Connection Failure**
1. Stop the Go backend
2. Observe browser console
3. Restart backend

**Expected Results:**
- ✅ Console shows reconnection attempts
- ✅ Maximum 5 reconnection attempts with 5-second intervals
- ✅ After backend restart, connection re-establishes
- ✅ No duplicate connections

**Test 8.3: Invalid Data**
1. Check browser console for any errors during normal operation

**Expected Results:**
- ✅ No console errors during normal operation
- ✅ Invalid SSE data is gracefully handled (logged, not crashed)

---

### Test 9: Multiple Notifications Flow

**Objective:** Test the system with multiple notifications.

**Steps:**
1. Execute 5 success jobs (wait for each to complete)
2. Execute 3 error jobs
3. Open notification dropdown

**Expected Results:**
- ✅ Dropdown shows latest 10 notifications
- ✅ Notifications are sorted by creation date (newest first)
- ✅ Badge shows correct unread count
- ✅ Mix of important and non-important notifications displayed correctly
- ✅ Scrolling works if more than 10 notifications

**Fix Multiple Errors:**
1. Click "最後のエラージョブを修正" button
2. Only the LAST error notification should be updated

**Expected Results:**
- ✅ Only the most recent error notification is updated
- ✅ Other error notifications remain unchanged
- ✅ Badge count updates correctly

---

### Test 10: Browser Refresh and State Persistence

**Objective:** Verify that state is properly loaded on page refresh.

**Steps:**
1. Create several notifications
2. Mark some as read
3. Refresh the browser (F5 or Cmd+R)

**Expected Results:**
- ✅ All notifications are loaded from backend
- ✅ Read/unread status is preserved
- ✅ Badge count is correct
- ✅ SSE connection is re-established
- ✅ New notifications continue to appear in real-time

---

## Performance Verification

### SSE Connection Stability
- ✅ Connection remains stable for extended periods (> 5 minutes)
- ✅ Keep-alive messages prevent timeout
- ✅ No memory leaks in browser (check DevTools Memory tab)

### API Response Times
- ✅ Notification creation: < 100ms
- ✅ Notification list fetch: < 200ms
- ✅ Read status update: < 100ms
- ✅ SSE event delivery: < 100ms after backend broadcast

---

## Cleanup

After testing, stop all services:

```bash
# Terminal 1 (Go backend)
Ctrl+C

# Terminal 2 (Next.js frontend)
Ctrl+C

# Stop PostgreSQL
cd examples/notification
docker compose down
```

---

## Common Issues and Troubleshooting

### Issue: SSE Connection Not Established
**Solution:**
- Check CORS settings in Go backend
- Verify backend is running on port 8080
- Check browser console for connection errors

### Issue: Notifications Not Appearing
**Solution:**
- Verify SSE connection in Network tab
- Check backend logs for notification creation
- Ensure user_id matches (EMP001)

### Issue: Badge Count Incorrect
**Solution:**
- Refresh the page to reload state
- Check read_at timestamps in database
- Verify mark-as-read API calls are successful

### Issue: Frontend Build Errors
**Solution:**
```bash
cd examples/notification/frontend
rm -rf .next node_modules
npm install
npm run dev
```

---

## Test Checklist Summary

Use this checklist to track your testing progress:

- [ ] Test 1: Success job and notification display
- [ ] Test 2: Error job and important notification
- [ ] Test 3: Fix job - update and read status
- [ ] Test 4: SSE connection and real-time updates
- [ ] Test 5: Read status management
- [ ] Test 6: Notification detail modal
- [ ] Test 7: Responsive design (mobile, tablet, desktop)
- [ ] Test 8: Error handling (backend down, SSE failure, invalid data)
- [ ] Test 9: Multiple notifications flow
- [ ] Test 10: Browser refresh and state persistence
- [ ] Performance verification
- [ ] Cleanup completed

---

## Success Criteria

All tests pass when:
1. ✅ All notifications appear in real-time via SSE
2. ✅ Read/unread status is correctly managed
3. ✅ Important notifications are properly highlighted
4. ✅ Fix job correctly updates error notifications
5. ✅ Responsive design works on all screen sizes
6. ✅ Error handling provides clear user feedback
7. ✅ No console errors during normal operation
8. ✅ SSE connection is stable and reconnects automatically
9. ✅ Badge count is always accurate
10. ✅ All UI interactions are smooth and responsive
