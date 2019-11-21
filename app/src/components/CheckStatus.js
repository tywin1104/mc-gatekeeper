import React from "react";
import { ListGroup, ListGroupItem, Container, Button } from "reactstrap";
import moment from "moment";
import RequestsService from "../service/RequestsService";
import i18next from "i18next";
import "./CheckStatus.css";

class CheckStatus extends React.Component {
  constructor(props) {
    super();
    this.state = {
      currentRequest: {},
      invalid: false
    };
  }
  componentDidMount() {
    const {
      match: { params }
    } = this.props;
    RequestsService.getRequestByEncodedID(params.id)
      .then(res => {
        if (res.status === 200) {
          this.setState({
            currentRequest: res.data.request
          });
        }
      })
      .catch(error => {
        this.setState({
          invalid: true
        });
      });
  }

  getButtonColor(status) {
    switch (status) {
      case "Approved":
        return "success";
      case "Denied":
        return "danger";
      default:
        return "info";
    }
  }

  getApplicationStatusText = status => {
    if (status === "Pending") {
      return i18next.t("Status.Pending");
    } else if (status === "Approved") {
      return i18next.t("Status.Approved");
    } else if (status === "Denied") {
      return i18next.t("Status.Denied");
    }
  };

  render() {
    let display;
    let currentRequest = this.state.currentRequest;
    if (!this.state.invalid && currentRequest) {
      display = (
        <Container>
          <ListGroup>
            <ListGroupItem active tag="a" href="#" action>
              {i18next.t("Status.Title")}
            </ListGroupItem>
            <ListGroupItem tag="a" action>
              <strong>{i18next.t("Status.Username")} </strong>{" "}
              {currentRequest.username}
            </ListGroupItem>
            <ListGroupItem tag="a" action>
              <strong>{i18next.t("Status.Email")} </strong>{" "}
              {currentRequest.email}
            </ListGroupItem>
            <ListGroupItem tag="a" action>
              <strong>{i18next.t("Status.Status")} </strong>
              <Button
                color={this.getButtonColor(currentRequest.status)}
                type="button"
              >
                {this.getApplicationStatusText(currentRequest.status)}
              </Button>
            </ListGroupItem>
            <ListGroupItem tag="a" action>
              <strong>{i18next.t("Status.ReferenceID")} </strong>{" "}
              {currentRequest._id}
            </ListGroupItem>
            <ListGroupItem disabled tag="a" href="#" action>
              <p>
                {i18next.t("Status.Submitted")}{" "}
                {moment
                  .parseZone(currentRequest.timestamp)
                  .local()
                  .fromNow()}
              </p>
              <p>{i18next.t("Status.Message")}</p>
            </ListGroupItem>
          </ListGroup>
        </Container>
      );
    } else {
      display = <h1>Invalid</h1>;
    }
    return <div>{display}</div>;
  }
}

export default CheckStatus;
