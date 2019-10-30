import axios from 'axios';

const API_HOST = window.REACT_APP_API_HOST ? window.REACT_APP_API_HOST : "";
console.log(API_HOST)

class RequestsService {

    // config: axios config containing auth bearer header
    getAllRequests(config) {
        return axios.get(`${API_HOST}/api/v1/internal/requests`, config)
    }

    approveRequestAdmin(requestID, config) {
        return axios.patch(`${API_HOST}/api/v1/internal/requests/${requestID}`, {
                status: "Approved",
                processedTimestamp: new Date().toISOString(),
                admin: 'admin' }, config)
    }

    denyRequestAdmin(requestID, config) {
        return axios.patch(`${API_HOST}/api/v1/internal/requests/${requestID}`, {
                status: "Denied",
                processedTimestamp: new Date().toISOString(),
                admin: 'admin' }, config)
    }
    ///////////////////////////////External API service call below///////////////////
    createRequest(data) {
        return axios.post(`${API_HOST}/api/v1/requests/`, data)
    }

    getRequestByEncodedID(encodedID) {
        return axios.get(`${API_HOST}/api/v1/requests/${encodedID}`)
    }

    approveRequest(requestID, admToken) {
        return axios.patch(`${API_HOST}/api/v1/requests/${requestID}?adm=${admToken}`, {
            status: "Approved",
            processedTimestamp: new Date(),
        })
    }

    denyRequest(requestID, admToken) {
        return axios.patch(`${API_HOST}/api/v1/requests/${requestID}?adm=${admToken}`, {
            status: "Denied",
            processedTimestamp: new Date(),
        })
    }

    // verify valid admin token first before displying any info in the action page
    verifyAdminToken(admToken) {
        return axios.get(`${API_HOST}/api/v1/verify?adm=${admToken}`)
    }
}

export default new RequestsService();