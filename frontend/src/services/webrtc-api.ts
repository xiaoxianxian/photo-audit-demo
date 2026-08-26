import axiosInstance from './api';

// ---------------------------------------------------------------------------
// WebRTC WHIP/WHEP signaling helpers (Phase 2)
// SDP bodies are raw text (application/sdp) — bypass the JSON unwrapping by
// using responseType 'text' and reading res.data directly.
// ---------------------------------------------------------------------------

/** Fetch the publisher's pending SDP offer without consuming it. */
export async function whepPeek(streamKey: string): Promise<string | null> {
  try {
    const res = await axiosInstance.get(`/webrtc/whep/${streamKey}`, { responseType: 'text' });
    return typeof res.data === 'string' ? res.data : null;
  } catch {
    return null; // 404 = no publisher
  }
}

/**
 * Complete the WHEP exchange: send the viewer's SDP answer, receive the
 * publisher's offer in return. Returns null when no publisher is live.
 */
export async function whepView(streamKey: string, answerSdp: string): Promise<string | null> {
  try {
    const res = await axiosInstance.post(`/webrtc/whep/${streamKey}`, answerSdp, {
      headers: { 'Content-Type': 'application/sdp' },
      responseType: 'text',
      timeout: 65000,
    });
    return typeof res.data === 'string' ? res.data : null;
  } catch {
    return null;
  }
}

/** Publisher: submit SDP offer and block until a viewer answers (or timeout). */
export async function whipPublish(streamKey: string, offerSdp: string): Promise<string | null> {
  try {
    const res = await axiosInstance.post(`/webrtc/whip/${streamKey}`, offerSdp, {
      headers: { 'Content-Type': 'application/sdp' },
      responseType: 'text',
      timeout: 65000,
    });
    return typeof res.data === 'string' ? res.data : null;
  } catch {
    return null;
  }
}

/** Publisher: tear down the signaling session. */
export async function whipDelete(streamKey: string): Promise<void> {
  try {
    await axiosInstance.delete(`/webrtc/whip/${streamKey}`);
  } catch {
    // best-effort
  }
}
