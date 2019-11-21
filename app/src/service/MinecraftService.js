import axios from "axios";

const API_HOST = process.env.REACT_APP_API_HOST
  ? process.env.REACT_APP_API_HOST
  : "";

class MinecraftService {
  getSkinImage(username) {
    return axios.get(`${API_HOST}/api/v1/minecraft/user/${username}/skin/`);
  }
  // External QR Code Service
  getQRCodeContent(imageURL) {
    return axios.get(
      `https://api.qrserver.com/v1/read-qr-code/?fileurl=${imageURL}`
    );
  }
}

export default new MinecraftService();
