import axios from "axios";

const API_HOST = process.env.REACT_APP_API_HOST
  ? process.env.REACT_APP_API_HOST
  : "";

class AuthService {
  login(credentials) {
    return axios.post(`${API_HOST}/api/v1/auth/`, credentials);
  }

  getAuthHeader() {
    return {
      headers: { Authorization: "Bearer " + this.getTokenInfo().value }
    };
  }

  getTokenInfo() {
    return JSON.parse(localStorage.getItem("token"));
  }
}

export default new AuthService();
