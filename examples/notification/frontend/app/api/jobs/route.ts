import { NextResponse } from 'next/server';

// メモリ内に最後のエラー通知IDを保存
let lastErrorNotificationId: number | null = null;

const BACKEND_API_URL = process.env.BACKEND_API_URL || 'http://localhost:8080';

interface JobRequest {
  type: 'success' | 'error' | 'fix';
}

export async function POST(request: Request) {
  try {
    const body = await request.json() as JobRequest;
    const { type } = body;

    // 型のバリデーション
    if (!type || !['success', 'error', 'fix'].includes(type)) {
      return NextResponse.json(
        { error: 'Invalid job type' },
        { status: 400 }
      );
    }

    // バックグラウンドでジョブを実行（非同期で実行し、待たない）
    executeJobInBackground(type).catch((error) => {
      console.error('Background job execution failed:', error);
    });

    // 即座に 202 Accepted を返す
    return NextResponse.json(
      { message: 'Job started' },
      { status: 202 }
    );
  } catch (error) {
    console.error('Error processing job request:', error);
    return NextResponse.json(
      { error: 'Failed to process job request' },
      { status: 500 }
    );
  }
}

async function executeJobInBackground(type: 'success' | 'error' | 'fix') {
  const userId = process.env.NEXT_PUBLIC_USER_ID || 'EMP001';

  if (type === 'fix') {
    // 修正ジョブ: 最後のエラー通知を更新（遅延なし）
    if (!lastErrorNotificationId) {
      console.error('No error notification to fix');
      return;
    }

    console.log(`[FIX JOB] Starting fix for notification ID: ${lastErrorNotificationId}`);
    console.log(`[FIX JOB] Backend URL: ${BACKEND_API_URL}`);

    try {
      const updateUrl = `${BACKEND_API_URL}/api/notifications/${lastErrorNotificationId}`;
      const updateBody = {
        title: 'Job Error (Resolved)',
        body: 'The issue has been resolved',
        important: false
      };

      console.log(`[FIX JOB] Sending PATCH to: ${updateUrl}`);
      console.log(`[FIX JOB] Request body:`, updateBody);

      // 通知を更新（タイトルと本文を修正内容に変更）
      const updateResponse = await fetch(updateUrl, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updateBody)
      });

      console.log(`[FIX JOB] Response status: ${updateResponse.status}`);

      if (!updateResponse.ok) {
        const errorText = await updateResponse.text();
        console.error(`[FIX JOB] Update failed: ${updateResponse.statusText}`, errorText);
        throw new Error(`Failed to update notification: ${updateResponse.statusText}`);
      }

      const responseData = await updateResponse.json();
      console.log('[FIX JOB] Update successful:', responseData);
      console.log('[FIX JOB] Error notification fixed (title and body updated, marked as unread)');
      
      // エラー通知IDをクリア
      lastErrorNotificationId = null;
    } catch (error) {
      console.error('[FIX JOB] Error fixing notification:', error);
    }
  } else if (type === 'success') {
    // 成功ジョブ: 3秒待機してから通知作成
    await new Promise(resolve => setTimeout(resolve, 3000));
    // 成功ジョブ: Go APIに成功通知を作成
    try {
      const response = await fetch(`${BACKEND_API_URL}/api/notifications`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: 'Job Completed',
          body: 'Job completed successfully',
          important: false,
          cancelable: false,
          user_ids: [userId]
        })
      });

      if (!response.ok) {
        throw new Error(`Failed to create success notification: ${response.statusText}`);
      }

      console.log('Success notification created');
    } catch (error) {
      console.error('Error creating success notification:', error);
    }
  } else if (type === 'error') {
    // エラージョブ: 3秒待機してからGo APIにエラー通知を作成
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    try {
      const response = await fetch(`${BACKEND_API_URL}/api/notifications`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: 'Job Error',
          body: 'An error occurred during job execution',
          important: true,
          cancelable: false,
          user_ids: [userId]
        })
      });

      if (!response.ok) {
        throw new Error(`Failed to create error notification: ${response.statusText}`);
      }

      const data = await response.json();
      // 通知IDをメモリに保存
      lastErrorNotificationId = data.id;
      console.log('Error notification created with ID:', lastErrorNotificationId);
    } catch (error) {
      console.error('Error creating error notification:', error);
    }
  }
}
