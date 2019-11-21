import React from "react";
import {
  ListGroup,
  ListGroupItem,
  Container,
  Button,
  Jumbotron,
  Alert,
  Form,
  FormGroup,
  Input
} from "reactstrap";
import moment from "moment";
import RequestsService from "../service/RequestsService";
import i18next from "i18next";
import "./AdminAction.css";

class AdminAction extends React.Component {
  constructor(props) {
    super();
    this.state = {
      currentRequest: {},
      invalid: false,
      adminToken: "",
      note: ""
    };
  }
  componentDidMount() {
    const search = this.props.location.search;
    const queryParams = new URLSearchParams(search);
    let adminToken = queryParams.get("adm");
    if (adminToken == null) {
      this.setState({
        invalid: true
      });
      return;
    } else {
      this.setState({
        adminToken: adminToken
      });
    }

    // Verify adminToken is valid
    const {
      match: { params }
    } = this.props;
    RequestsService.verifyAdminToken(params.id, adminToken).catch(error => {
      this.setState({
        invalid: true
      });
      return;
    });
    // Get current request related to this specific email ticket?
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
        return;
      });
  }

  handleInputChange = event => {
    const { value, name } = event.target;
    this.setState({
      [name]: value
    });
  };

  onDecision = (event, newStatus) => {
    event.preventDefault();
    const {
      match: { params }
    } = this.props;
    let note = this.state.note;
    let promise;
    if (newStatus === "Denied") {
      promise = RequestsService.denyRequest(
        params.id,
        this.state.adminToken,
        note
      );
    } else {
      promise = RequestsService.approveRequest(
        params.id,
        this.state.adminToken,
        note
      );
    }
    promise
      .then(res => {
        if (res.status === 200) {
          alert(i18next.t("Action.CompletedMsg"));
          window.location.reload();
        }
      })
      .catch(error => {
        if (error.response) {
          if (error.response.status === 400) {
            alert(i18next.t("Action.InvalidTokenErrMsg"));
          } else {
            alert(i18next.t("Action.InternalErrMsg"));
          }
        }
      });
  };

  render() {
    let display;
    let currentRequest = this.state.currentRequest;
    if (
      !this.state.invalid &&
      currentRequest &&
      this.state.currentRequest.status === "Pending"
    ) {
      display = (
        <Container>
          <ListGroup>
            <ListGroupItem active action>
              {i18next.t("Action.Title")}
            </ListGroupItem>
            <ListGroupItem action>
              <strong>{i18next.t("Action.Gender")}</strong>{" "}
              {currentRequest.gender}
            </ListGroupItem>
            <ListGroupItem action>
              <strong>{i18next.t("Action.Age")}</strong> {currentRequest.age}
            </ListGroupItem>
            <ListGroupItem action>
              <strong>{i18next.t("Action.ApplicationText")}</strong>
              <Jumbotron>
                <p>{currentRequest.info.applicationText}</p>
              </Jumbotron>
            </ListGroupItem>
            <ListGroupItem disabled action>
              {i18next.t("Action.Submitted")}{" "}
              {moment
                .parseZone(currentRequest.timestamp)
                .local()
                .fromNow()}
            </ListGroupItem>
          </ListGroup>
          <Form>
            <FormGroup>
              <Input
                type="textarea"
                name="note"
                placeholder={i18next.t("Action.NotePlaceHolder")}
                value={this.state.note}
                onChange={this.handleInputChange}
              />
            </FormGroup>
          </Form>
          <Button
            className="actionButton"
            onClick={e => this.onDecision(e, "Approved")}
            color="success"
            outline
            size="lg"
            type="button"
          >
            {i18next.t("Action.Approve")}
          </Button>
          <Button
            className="actionButton"
            onClick={e => this.onDecision(e, "Denied")}
            color="danger"
            outline
            size="lg"
            type="button"
          >
            {i18next.t("Action.Deny")}
          </Button>

          <Jumbotron fluid>
            <Container fluid>
              <h3>{i18next.t("Action.NoteTitle")}</h3>
              <p className="lead">{i18next.t("Action.NoteContent")}</p>
            </Container>
          </Jumbotron>
        </Container>
      );
    } else if (
      currentRequest &&
      (currentRequest.status === "Approved" ||
        currentRequest.status === "Denied")
    ) {
      display = (
        <div>
          <Alert color="info">{i18next.t("Action.FulfilledMsg")}</Alert>
        </div>
      );
    } else if (this.state.invalid) {
      display = <p>Invalid route</p>;
    }
    return <div>{display}</div>;
  }
}

export default AdminAction;
