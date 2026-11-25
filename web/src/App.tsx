
import { useState } from 'react'
import { searchTrips } from './api/trips'
import type { TripOption } from './types/trip'
import './App.css'

function formatPrice(price: number) {
  return new Intl.NumberFormat('ru-RU', {
    style: 'currency',
    currency: 'RUB',
    maximumFractionDigits: 0,
  }).format(price / 100)
}

// Не используем Date для преобразования часового пояса.
// departure_at содержит локальное время аэропорта.
function formatDateTime(value: string) {
  const [date, time] = value.split('T')
  const [year, month, day] = date.split('-')

  return `${day}.${month}.${year} ${time.slice(0, 5)}`
}


function TripCard({ trip }: { trip: TripOption }) {
  const { outbound, inbound } = trip

  return (
    <article className="trip-card">
      <div className="trip-card-header">
        <span>Поездка на {trip.trip_duration} дней</span>
        <strong>{formatPrice(trip.total_price)}</strong>
      </div>

      {/* Билет туда */}
      <div className="flight-row">
        <div className="flight-direction">
          <span className="flight-label">Туда</span>
          <strong>{outbound.origin}</strong>
          <span>{formatDateTime(outbound.departure_at)}</span>
          <small>
            {outbound.origin_airport} → {outbound.destination_airport}
          </small>
        </div>

        <div className="flight-info">
          <strong>
            {outbound.airline} {outbound.flight_number}
          </strong>
          <span>{outbound.transfers} пересадок</span>
          <span>
            {Math.floor(outbound.duration / 60)} ч {outbound.duration % 60} мин
          </span>
          <span>
            Стоимость: {formatPrice(outbound.price)}
          </span>
          <small>
            ID в БД: {outbound.id}
          </small>
        </div>

        <div className="flight-direction">
          <span className="flight-label">Прилёт</span>
          <strong>{outbound.destination}</strong>
          <small>{outbound.destination_airport}</small>
        </div>
      </div>

      <div className="flight-divider" />

      {/* Билет обратно */}
      <div className="flight-row">
        <div className="flight-direction">
          <span className="flight-label">Обратно</span>
          <strong>{inbound.origin}</strong>
          <span>{formatDateTime(inbound.departure_at)}</span>
          <small>
            {inbound.origin_airport} → {inbound.destination_airport}
          </small>
        </div>

        <div className="flight-info">
          <strong>
            {inbound.airline} {inbound.flight_number}
          </strong>
          <span>{inbound.transfers} пересадок</span>
          <span>
            {Math.floor(inbound.duration / 60)} ч {inbound.duration % 60} мин
          </span>
          <span>
            Стоимость: {formatPrice(inbound.price)}
          </span>
          <small>
            ID в БД: {inbound.id}
            </small>
        </div>

        <div className="flight-direction">
          <span className="flight-label">Прилёт</span>
          <strong>{inbound.destination}</strong>
          <small>{inbound.destination_airport}</small>
        </div>
      </div>

      {/* Ссылки на покупку */}
      <div className="booking-buttons">
        {outbound.booking_link && (
          <a
            className="booking-button"
            href={outbound.booking_link}
            target="_blank"
            rel="noreferrer"
          >
            Купить билет туда
          </a>
        )}

        {inbound.booking_link && (
          <a
            className="booking-button"
            href={inbound.booking_link}
            target="_blank"
            rel="noreferrer"
          >
            Купить билет обратно
          </a>
        )}
      </div>
    </article>
  )
}

function App() {
  const [origin, setOrigin] = useState('MOW')
  const [destinations, setDestinations] = useState('SHA, HGH')
  const [returnOrigins, setReturnOrigins] = useState('SHA, HGH, TYO')

  const [departureFrom, setDepartureFrom] = useState('2027-01-01')
  const [departureTo, setDepartureTo] = useState('2027-03-31')

  const [minDuration, setMinDuration] = useState(7)
  const [maxDuration, setMaxDuration] = useState(9)
  const [maxTransfers, setMaxTransfers] = useState(1)

  const [trips, setTrips] = useState<TripOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSearch(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    setLoading(true)
    setError('')
    setTrips([])

    try {
      const result = await searchTrips({
        origin: origin.trim().toUpperCase(),
        destinations: destinations.split(',').map((x) => x.trim().toUpperCase()).filter(Boolean),
        return_origins: returnOrigins.split(',').map((x) => x.trim().toUpperCase()).filter(Boolean),
        departure_from: departureFrom,
        departure_to: departureTo,
        min_trip_duration: minDuration,
        max_trip_duration: maxDuration,
        max_transfers: maxTransfers,
        page: 1,
        limit: 100,
      })

      setTrips(result.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка поиска')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="app">
      <header className="header">
        <h1>Avia Bro</h1>
        <span>Поиск авиабилетов</span>
      </header>

      <section className="search-section">
        <h2>Найти поездку</h2>

        <form onSubmit={handleSearch}>
          <div className="form-grid">
            <label>
              Город вылета
              <input
                value={origin}
                onChange={(e) => setOrigin(e.target.value)}
                required
              />
            </label>

            <label>
              Города назначения
              <input
                value={destinations}
                onChange={(e) => setDestinations(e.target.value)}
                placeholder="SHA, HGH"
                required
              />
            </label>

            <label>
              Города возвращения
              <input
                value={returnOrigins}
                onChange={(e) => setReturnOrigins(e.target.value)}
                placeholder="SHA, HGH, TYO"
                required
              />
            </label>

            <label>
              Вылет с
              <input
                type="date"
                value={departureFrom}
                onChange={(e) => setDepartureFrom(e.target.value)}
                required
              />
            </label>

            <label>
              Вылет по
              <input
                type="date"
                value={departureTo}
                onChange={(e) => setDepartureTo(e.target.value)}
                required
              />
            </label>

            <label>
              Минимум дней
              <input
                type="number"
                min="1"
                value={minDuration}
                onChange={(e) => setMinDuration(Number(e.target.value))}
              />
            </label>

            <label>
              Максимум дней
              <input
                type="number"
                min="1"
                value={maxDuration}
                onChange={(e) => setMaxDuration(Number(e.target.value))}
              />
            </label>

            <label>
              Максимум пересадок
              <select
                value={maxTransfers}
                onChange={(e) => setMaxTransfers(Number(e.target.value))}
              >
                <option value={0}>Без пересадок</option>
                <option value={1}>До 1</option>
                <option value={2}>До 2</option>
                <option value={3}>До 3</option>
              </select>
            </label>
          </div>

          <button className="search-button" type="submit" disabled={loading}>
            {loading ? 'Ищем билеты...' : 'Найти билеты'}
          </button>
        </form>
      </section>

      <section className="results-section">
        <div className="results-header">
          <h2>Результаты поиска</h2>
          <span>{trips.length} вариантов</span>
        </div>

        {error && <p className="error-message">{error}</p>}

        {!loading && !error && trips.length === 0 && (
          <p className="empty-message">Укажите параметры и запустите поиск.</p>
        )}

        {trips.map((trip) => (
          <TripCard
            key={`${trip.outbound.flight_key}_${trip.inbound.flight_key}`}
            trip={trip}
          />
        ))}
      </section>
    </main>
  )
}

export default App
