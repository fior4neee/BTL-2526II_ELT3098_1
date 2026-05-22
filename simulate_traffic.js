/**
 * VNU-LEO Traffic Simulator
 *
 * Mô phỏng phản ánh đúng thực tế:
 * - Mỗi router connect từ vị trí địa lý thực của mình
 * - Chỉ handover đến gateway gần nhất theo tọa độ (không vi phạm geofence)
 * - Router Mobile (004) có thể handover tự do qua toàn quốc
 * - Router Fixed chỉ handover trong phạm vi 50 km từ home location
 */

const API_BASE = 'http://localhost:8081/api/v1';

// ─── Cấu hình router theo đúng billing subscription trong core_network/main.go ───
// Mỗi router có: home location thực, danh sách gateway được phép, và MAC thực từ devices.json
const ROUTER_CONFIGS = [
    {
        id:   'router-vnu-leo-001',
        mac:  '00:1B:44:11:3A:B7',
        plan: 'fixed',
        home: { lat: 21.0285, lon: 105.8542, alt: 0 },  // Hà Nội
        // Chỉ được connect gateway trong vòng 50 km → GW-HAN-01 (Hà Nội)
        allowedGateways: ['GW-HAN-01'],
        // Location để trigger handover: vẫn phải trong 50 km của home (Hà Nội)
        handoverLocs: [
            { lat: 21.0285, lon: 105.8542, alt: 0 }, // Hà Nội (home)
            { lat: 21.0500, lon: 105.8000, alt: 0 }, // Gần Hà Nội
        ],
    },
    {
        id:   'router-vnu-leo-002',
        mac:  '3C:A8:2A:FF:11:0C',
        plan: 'fixed',
        home: { lat: 21.0285, lon: 105.8542, alt: 0 },  // Hà Nội
        allowedGateways: ['GW-HAN-01'],
        handoverLocs: [
            { lat: 21.0285, lon: 105.8542, alt: 0 },
            { lat: 21.0600, lon: 105.8800, alt: 0 },
        ],
    },
    {
        id:   'router-vnu-leo-004',
        mac:  '54:E1:AD:99:3B:1F',
        plan: 'mobile',
        home: { lat: 20.8449, lon: 106.6881, alt: 0 },  // Hải Phòng
        // Mobile plan → có thể dùng bất kỳ gateway nào
        allowedGateways: ['GW-HAN-01', 'GW-DAN-01', 'GW-HCM-01'],
        handoverLocs: [
            { lat: 21.0285, lon: 105.8542, alt: 0 }, // Hà Nội (GW-HAN-01)
            { lat: 16.0471, lon: 108.2062, alt: 0 }, // Đà Nẵng (GW-DAN-01)
            { lat: 10.7626, lon: 106.6601, alt: 0 }, // HCMC (GW-HCM-01)
        ],
    },
    {
        id:   'router-vnu-leo-005',
        mac:  'F8:1A:67:B2:EE:90',
        plan: 'fixed',
        home: { lat: 10.0452, lon: 105.7469, alt: 0 },  // Cần Thơ (~70 km từ HCMC)
        // Cần Thơ cách HCMC ~70 km → ngoài giới hạn 50 km, nhưng gần nhất
        // → chỉ connect GW-HCM-01 với location là home location (Cần Thơ, <50km từ home)
        allowedGateways: ['GW-HCM-01'],
        handoverLocs: [
            { lat: 10.0452, lon: 105.7469, alt: 0 }, // Cần Thơ (home)
            { lat: 10.0600, lon: 105.8000, alt: 0 }, // Gần Cần Thơ
        ],
    },
    {
        id:   'router-vnu-leo-006',
        mac:  '00:22:44:66:88:AA',
        plan: 'fixed',
        home: { lat: 12.2388, lon: 109.1967, alt: 0 },  // Nha Trang
        // Nha Trang gần GW-DAN-01 (Đà Nẵng, ~350 km) và GW-HCM-01 (HCMC, ~440 km)
        // Đều > 50 km → thực ra sẽ bị chặn hoàn toàn nếu geofence nghiêm ngặt
        // Thử connect và ghi nhận kết quả thực
        allowedGateways: ['GW-DAN-01'],
        handoverLocs: [
            { lat: 12.2388, lon: 109.1967, alt: 0 }, // Nha Trang (home)
        ],
    },
];

// ─── Helpers ───────────────────────────────────────────────────────────────────

async function fetchJson(path, options = {}) {
    const res = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers: { 'Content-Type': 'application/json', ...(options.headers ?? {}) },
    });
    const text = await res.text();
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${text}`);
    return JSON.parse(text);
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

function pick(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
}

// ─── Main simulation ───────────────────────────────────────────────────────────

async function run() {
    console.log('🛰️  VNU-LEO Traffic Simulator — Realistic Mode');
    console.log('─'.repeat(55));

    // 1. Kiểm tra API backend đang chạy
    try {
        await fetchJson('/health');
        console.log('✅ Core Network API reachable at', API_BASE);
    } catch {
        console.error('❌ Cannot reach Core Network. Make sure core_network is running on port 8081.');
        process.exit(1);
    }

    // 2. Fetch gateway list thực từ API
    const gwResp = await fetchJson('/gateways');
    const gateways = gwResp.data ?? gwResp.gateways ?? [];
    const gwById = Object.fromEntries(gateways.map(g => [g.id, g]));
    console.log(`✅ Found ${gateways.length} gateways: ${gateways.map(g => g.name).join(', ')}`);
    console.log('');

    // 3. Tạo sessions — mỗi router connect từ home location thực của mình
    console.log('📡 Connecting routers to their nearest gateways...');
    const activeSessions = []; // { session, config }

    for (const config of ROUTER_CONFIGS) {
        try {
            const data = await fetchJson('/router/connect', {
                method: 'POST',
                body: JSON.stringify({
                    device_id: config.id,
                    router_mac: config.mac,
                    location: config.home,
                }),
            });
            const session = data.session;
            activeSessions.push({ session, config });
            console.log(`  ✅ [${config.plan.toUpperCase()}] ${config.id} → ${session.current_gateway_id} (Session: ${session.session_id})`);
        } catch (e) {
            console.warn(`  ⚠️  ${config.id} connect failed: ${e.message}`);
        }
    }

    if (activeSessions.length === 0) {
        console.error('\n❌ No sessions established. Cannot simulate handovers.');
        process.exit(1);
    }

    console.log(`\n✅ ${activeSessions.length} session(s) established.`);
    console.log('\n🔄 Starting handover simulation (every 6 seconds)...');
    console.log('   Each router only handovers within its allowed gateways.');
    console.log('   Press Ctrl+C to stop.\n');

    // 4. Vòng lặp handover liên tục — mỗi router chọn gateway được phép theo địa lý
    let tick = 0;
    setInterval(async () => {
        tick++;
        if (activeSessions.length === 0) return;

        // Chọn một session ngẫu nhiên để trigger handover
        const { session, config } = pick(activeSessions);

        // Chỉ handover đến gateway khác trong danh sách được phép của router này
        const candidates = config.allowedGateways.filter(gwId => {
            // Nếu chỉ có 1 gateway được phép → vẫn trigger để sinh handover history
            return config.allowedGateways.length === 1 || gwId !== session.current_gateway_id;
        });
        const targetGwId = pick(candidates);
        const targetGw = gwById[targetGwId];
        if (!targetGw) return;

        // Chọn location tương ứng với gateway đích (phải trong geofence của router)
        // Với mobile router → dùng location của gateway đích
        // Với fixed router → dùng home location (vẫn nằm trong geofence)
        const handoverLoc = config.plan === 'mobile'
            ? pick(config.handoverLocs)
            : config.home;

        try {
            await fetchJson('/handover/trigger', {
                method: 'POST',
                body: JSON.stringify({
                    session_id: session.session_id,
                    target_gateway_id: targetGwId,
                    location: handoverLoc,
                }),
            });
            // Cập nhật state nội bộ simulator
            session.current_gateway_id = targetGwId;
            const arrow = `${targetGwId.padEnd(12)}`;
            console.log(`  [T+${String(tick * 6).padStart(4)}s] ✅ ${config.id.slice(-3)} → ${arrow} (${session.session_id})`);
        } catch (e) {
            console.warn(`  [T+${String(tick * 6).padStart(4)}s] ⚠️  ${config.id.slice(-3)} handover failed: ${e.message.slice(0, 80)}`);
        }
    }, 6000);
}

run().catch(err => {
    console.error('Fatal error:', err);
    process.exit(1);
});
