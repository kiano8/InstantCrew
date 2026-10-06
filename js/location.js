// js/location.js — geolocator for Instant Crew
(function () {
  'use strict';

  const GEO_OPTIONS = { enableHighAccuracy: true, timeout: 10000, maximumAge: 60000 };

  const ERROR_MESSAGES = {
    UNSUPPORTED: 'Location is not supported by this browser.',
    PERMISSION_DENIED: 'Location permission was denied. Please allow access or enter the address manually.',
    POSITION_UNAVAILABLE: 'Your location could not be determined. Please enter it manually.',
    TIMEOUT: 'Location request timed out. Please try again.',
    UNKNOWN: 'Something went wrong while getting your location.',
  };

  function makeError(code) {
    return { code, message: ERROR_MESSAGES[code] || ERROR_MESSAGES.UNKNOWN };
  }

  /** Promise wrapper around navigator.geolocation → { lat, lng, accuracy } */
  function getCurrentPosition(options = {}) {
    return new Promise((resolve, reject) => {
      if (!('geolocation' in navigator)) {
        reject(makeError('UNSUPPORTED'));
        return;
      }
      navigator.geolocation.getCurrentPosition(
        (pos) => resolve({
          lat: pos.coords.latitude,
          lng: pos.coords.longitude,
          accuracy: pos.coords.accuracy,
        }),
        (err) => {
          const map = { 1: 'PERMISSION_DENIED', 2: 'POSITION_UNAVAILABLE', 3: 'TIMEOUT' };
          reject(makeError(map[err.code] || 'UNKNOWN'));
        },
        { ...GEO_OPTIONS, ...options }
      );
    });
  }

  /**
   * Reverse geocode via OpenStreetMap Nominatim (free, no API key).
   * Returns { city, address } or null on failure (coordinates are still usable).
   * NOTE: fine for demos/low traffic. For production use your own geocoder
   * (Google Geocoding API, Mapbox, BigDataCloud, etc.) — Nominatim limits to ~1 req/sec.
   */
  async function reverseGeocode(lat, lng) {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 6000);
    try {
      const url = `https://nominatim.openstreetmap.org/reverse?format=jsonv2&addressdetails=1&zoom=18&lat=${lat}&lon=${lng}`;
      const res = await fetch(url, { signal: ctrl.signal, headers: { Accept: 'application/json' } });
      if (!res.ok) return null;
      const data = await res.json();
      const a = data.address || {};

      const city = a.city || a.town || a.municipality || a.city_district || a.county || a.state_district || '';
      const place = a.amenity || a.building || a.shop || a.road || '';
      const area = a.suburb || a.neighbourhood || a.quarter || a.village || '';
      const address = [place, area, city].filter(Boolean).join(', ') || data.display_name || '';

      return { city, address };
    } catch (e) {
      return null;
    } finally {
      clearTimeout(timer);
    }
  }

  function buildMapsUrl(lat, lng) {
    return `https://www.google.com/maps/search/?api=1&query=${lat},${lng}`;
  }

  /** One call: permission prompt → GPS → reverse geocode → everything the UI needs. */
  async function locate(options) {
    const { lat, lng, accuracy } = await getCurrentPosition(options);
    const geo = await reverseGeocode(lat, lng);
    return {
      lat,
      lng,
      accuracy,
      city: geo ? geo.city : '',
      address: geo ? geo.address : '',
      label: (geo && geo.address) || `${lat.toFixed(5)}, ${lng.toFixed(5)}`,
      mapsUrl: buildMapsUrl(lat, lng), // exact pin, even if the label is just an address
    };
  }

  window.InstantCrewLocation = { getCurrentPosition, reverseGeocode, buildMapsUrl, locate };
})();
