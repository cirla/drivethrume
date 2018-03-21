module Main exposing (..)

import Date exposing (Date)
import Date.Extra.Config.Config_en_us exposing (config)
import Date.Extra.Core exposing (fromTime)
import Date.Extra.Format as Format exposing (format)
import Dict exposing (Dict)
import Geolocation exposing (Location)
import Html exposing (Html, a, button, div, i, li, nav, p, span, strong, text, ul)
import Html.Attributes exposing (attribute, class, id, href, type_)
import Html.Attributes.Aria exposing (role)
import Http exposing (Body, Request, expectJson, header, jsonBody, request)
import Json.Decode exposing (Decoder, at, bool, field, float, list, maybe, string, succeed)
import Json.Decode.Extra exposing ((|:), date)
import Json.Encode as Json exposing (encode)
import Navigation
import Task
import UrlParser as Url exposing (Parser, (</>), oneOf, s, top)


-- PROGRAM


main : Program Never Model Msg
main =
  Navigation.program UrlChange
    { init = init
    , view = view
    , update = update
    , subscriptions = subscriptions
    }


-- NAVIGATION


type Route
  = Home
  | About


toHash : Route -> String
toHash route =
  case route of
    Home ->
      "#"
    About ->
      "#about"


route : Url.Parser (Route -> a) a
route =
  Url.oneOf
    [ Url.map Home top
    , Url.map About (s "about")
    ]


parseRoute : Navigation.Location -> Maybe Route
parseRoute =
  Url.parseHash route


-- MODEL


type alias DriveThru =
  { type_: String
  , address: String
  , lat: Float
  , lng: Float
  , distanceMiles: Float
  , isOpen: Bool
  , openTime: Maybe Date
  , closeTime: Maybe Date
  }


type alias LocationState =
  Result Geolocation.Error (Maybe Location)


type alias DriveThrusState =
  Result Http.Error (Maybe (List DriveThru))


type alias Model =
  { route: Maybe Route
  , loc: LocationState
  , driveThrus: DriveThrusState
  }


initialModel : Model
initialModel =
  { route = Nothing
  , loc = Ok Nothing
  , driveThrus = Ok Nothing
  }


init : Navigation.Location -> ( Model, Cmd Msg )
init location =
  ( { initialModel | route = parseRoute location }
  , Task.attempt UpdateLoc Geolocation.now
  )


-- UPDATE


type Msg
  = UrlChange Navigation.Location
  | UpdateLoc (Result Geolocation.Error Location)
  | UpdateDriveThrus (Result Http.Error (List DriveThru))


update : Msg -> Model -> (Model, Cmd Msg)
update msg model =
  case msg of
    UrlChange location ->
      ( { model | route = parseRoute location }
      , Cmd.none
      )
    UpdateLoc result ->
      ( { model | loc = Result.map Just result }
      , case result of
          Ok loc -> queryApi loc
          _ -> Cmd.none
      )
    UpdateDriveThrus result ->
      ( { model | driveThrus = Result.map Just result }
      , Cmd.none
      )



-- HTTP


apiUrl : String
apiUrl = "/.netlify/functions/find_drivethrus"


apiBody : Float -> Float -> Body
apiBody lat lng =
  Json.object
    [ ("lat", Json.float lat)
    , ("lng", Json.float lng)
    , ("show_closed", Json.bool True)
    ]
  |> jsonBody


queryApi : Location -> Cmd Msg
queryApi loc =
  let
    body = apiBody loc.latitude loc.longitude
  in
    Http.send UpdateDriveThrus (Http.post apiUrl body decodeResponse)


decodeResponse : Decoder (List DriveThru)
decodeResponse =
  let decodeDriveThru =
    succeed DriveThru
      |: (field "type" string)
      |: (field "address" string)
      |: (field "lat" float)
      |: (field "lng" float)
      |: (field "distance_miles" float)
      |: (field "is_open" bool)
      |: (field "open_time" <| maybe date)
      |: (field "close_time" <| maybe date)
  in
    at ["locations"] <| list decodeDriveThru


-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
  Geolocation.changes (UpdateLoc << Ok)


-- VIEW


view : Model -> Html Msg
view model =
  div []
    [ viewNav model
    , div [ class "container" ] [ viewRoute model ]
    ]


viewNav : Model -> Html Msg
viewNav model =
  nav [ class "navbar fixed-top navbar-expand-md navbar-dark bg-dark" ]
    [ a [ class "navbar-brand", href "#" ] [ text "Drive-Thru Me" ]
    , button
        [ class "navbar-toggler"
        , type_ "button"
        , attribute "data-toggle" "collapse"
        , attribute "data-target" "#navbarSupportedContent"
        ]
        [ span [ class "navbar-toggler-icon" ] [] ]
    , div [ class "collapse navbar-collapse", id "navbarSupportedContent" ]
      [ ul [ class "navbar-nav mr-auto" ]
        [ navLink Home "Home" (model.route == Just Home)
        , navLink About "About" (model.route == Just About)
        ]
      ]
    ]


navLink : Route -> String -> Bool -> Html msg
navLink route title active =
  li [ "nav-item" ++ (if active then " active" else "") |> class ] [
    a [ class "nav-link", href (toHash route) ] [ text title ] ]


viewRoute : Model -> Html Msg
viewRoute model =
  case model.route of
    Just About -> viewAbout model
    _ -> viewHome model


viewHome : Model -> Html Msg
viewHome model =
  case model.loc of
    Ok (Just loc) -> viewDriveThrus model.driveThrus
    Ok Nothing -> div [ class "alert alert-primary" ] [ text "Location not available" ]
    Err e -> viewError "Error determining location." e


viewDriveThrus : DriveThrusState -> Html Msg
viewDriveThrus driveThrus =
  case driveThrus of
    Ok (Just []) -> div [ class "alert alert-info" ] [ text "No Nearby Drive-Thrus Found." ]
    Ok (Just dts) -> ul [ class "list-group" ] <| List.map viewDriveThru dts
    Ok Nothing -> div [] [ text "Loading..." ]
    Err e -> viewError "Error loading drive-thru locations." e


typeNameDict : Dict String String
typeNameDict = Dict.fromList
  [ ("mcdonalds", "McDonald's")
  ]


viewDriveThru : DriveThru -> Html Msg
viewDriveThru d =
  li [ class <| "list-group-item" ++ (if d.isOpen then "" else " list-group-item-danger") ]
    [ p [ class "h3" ] [ text <| Maybe.withDefault "" (Dict.get d.type_ typeNameDict) ]
    , p [] [ text d.address ]
    , p [] [ text <| (toString d.distanceMiles) ++ " miles away" ]
    , p []
      [ case d.isOpen of
          True ->
            case d.closeTime of
              Nothing -> text "Open 24 Hours"
              Just time -> text <| "Closes at " ++ (format config "%H:%M" time)
          False ->
            strong [] [ text <| "Closed until " ++ (format config "%H:%M" <| Maybe.withDefault (fromTime 0) d.openTime) ]
      ]
    ]


viewError : String -> a -> Html msg
viewError msg e =
  let
    _ = Debug.log "Error" <| toString e
  in
    div [ class "alert alert-danger", role "alert" ] [ text msg ]


viewAbout : Model -> Html Msg
viewAbout model =
  div []
    [ p [] [ text "Hungry but don't want to get out of the car?" ]
    , p [] [ text "Whether it's just too cold or you don't want to deal with the carseat battle for the eighth time today, we've got you covered." ]
    , p [] 
      [ text "Are we missing support for your favorite drive-thru? "
      , a [ href "https://github.com/cirla/drivethrume/issues" ] [ text "Open an issue " ]
      , text "(or submit a pull request) "
      , a [ href "https://github.com/cirla/drivethrume" ] [ text "on GitHub" ]
      , text "!"
      ]
    ]

