import axios from "axios";

const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const API = axios.create({
  baseURL: apiUrl,
  headers: { "Content-Type": "application/json" },
});

export default API;
