import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const BASE_URL = (__ENV.BASE_URL || 'http://localhost:8080') + (__ENV.API_PREFIX || '');

const VUS = Number(__ENV.VUS || 5);
const RAMP_UP = __ENV.RAMP_UP || '15s';
const DURATION = __ENV.DURATION || '1m';
const RAMP_DOWN = __ENV.RAMP_DOWN || '15s';
const MAX_LOGGED_ERRORS = Number(__ENV.MAX_LOGGED_ERRORS || 25);

const usersCreated = new Counter('load_users_created');
const roomsCreated = new Counter('load_rooms_created');
const bookingsCreated = new Counter('load_bookings_created');
const flowOk = new Rate('load_flow_ok');

export const options = {
  scenarios: {
    slotify: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: RAMP_UP, target: VUS },   // плавный разгон
        { duration: DURATION, target: VUS },  // рабочая нагрузка
        { duration: RAMP_DOWN, target: 0 },   // затухание
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],    // <5% ошибок
    http_req_duration: ['p(95)<750'],  // 95% запросов быстрее 750мс
  },
};

const JSON_HDR = { headers: { 'Content-Type': 'application/json' } };
const rnd = (n) => Math.floor(Math.random() * n);

let loggedErrors = 0;

function fail(res, label) {
  if (loggedErrors < MAX_LOGGED_ERRORS) {
    loggedErrors++;
    console.error(`${label}: статус ${res.status} — ${res.body}`);
  }
}

function createUser(role) {
  const res = http.post(`${BASE_URL}/users`, JSON.stringify({ role }), JSON_HDR);
  const created = check(res, { [`пользователь ${role} создан (201)`]: (r) => r.status === 201 });
  if (!created) {
    fail(res, `POST /users (${role})`);
    return null;
  }
  usersCreated.add(1);
  return res.json('token');
}

function authHdr(token) {
  return { headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` } };
}

export function setup() {
  const health = http.get(`${BASE_URL}/healthz`);
  if (health.status !== 200) {
    throw new Error(`Приложение недоступно: GET ${BASE_URL}/healthz -> ${health.status}`);
  }
}

export default function () {
  const adminToken = createUser('admin');
  const memberToken = createUser('member');

  if (!adminToken || !memberToken) {
    flowOk.add(false);
    sleep(1);
    return;
  }

  const capacity = 2 + rnd(7);
  const spaceRes = http.post(
    `${BASE_URL}/spaces`,
    JSON.stringify({
      name: `k6-room-${Date.now() % 100000}-${rnd(1000)}`,
      type: Math.random() < 0.5 ? 'meeting_room' : 'desk',
      capacity,
    }),
    authHdr(adminToken),
  );
  check(spaceRes, { 'комната создана (201)': (r) => r.status === 201 });
  const spaceId = spaceRes.status === 201 ? spaceRes.json('id') : null;
  if (!spaceId) fail(spaceRes, 'POST /spaces');
  else roomsCreated.add(1);

  let booked = false;
  if (spaceId) {
    const date = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
    const start = 108 + rnd(80);
    const end = start + 6 + rnd(18);
    const bookRes = http.post(
      `${BASE_URL}/bookings`,
      JSON.stringify({
        space_id: spaceId,
        date,
        start_slot: start,
        end_slot: end,
        people: 1 + rnd(capacity), 
      }),
      authHdr(memberToken),
    );
    booked = check(bookRes, { 'слоты забронированы (201)': (r) => r.status === 201 });
    if (!booked) fail(bookRes, 'POST /bookings');
    else bookingsCreated.add(1);

    const my = http.get(`${BASE_URL}/bookings/my`, authHdr(memberToken));
    const okMy = check(my, { 'список своих броней (200)': (r) => r.status === 200 });
    if (!okMy) fail(my, 'GET /bookings/my');
  }

  flowOk.add(!!spaceId && booked);
  sleep(1);
}