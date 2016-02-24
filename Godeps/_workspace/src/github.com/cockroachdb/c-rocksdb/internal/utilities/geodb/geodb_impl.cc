//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#ifndef ROCKSDB_LITE

#include "utilities/geodb/geodb_impl.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <vector>
#include <map>
#include <string>
#include <limits>
#include "db/filename.h"
#include "util/coding.h"
#include "util/string_util.h"

//
// There are two types of keys. The first type of key-values
// maps a geo location to the set of object ids and their values.
// Table 1
//   key     : p + : + $quadkey + : + $id +
//             : + $latitude + : + $longitude
//   value  :  value of the object
// This table can be used to find all objects that reside near
// a specified geolocation.
//
// Table 2
//   key  : 'k' + : + $id
//   value:  $quadkey

namespace rocksdb {

const double GeoDBImpl::PI = 3.141592653589793;
const double GeoDBImpl::EarthRadius = 6378137;
const double GeoDBImpl::MinLatitude = -85.05112878;
const double GeoDBImpl::MaxLatitude = 85.05112878;
const double GeoDBImpl::MinLongitude = -180;
const double GeoDBImpl::MaxLongitude = 180;

GeoDBImpl::GeoDBImpl(DB* db, const GeoDBOptions& options) :
  GeoDB(db, options), db_(db), options_(options) {
}

GeoDBImpl::~GeoDBImpl() {
}

Status GeoDBImpl::Insert(const GeoObject& obj) {
  WriteBatch batch;

  // It is possible that this id is already associated with
  // with a different position. We first have to remove that
  // association before we can insert the new one.

  // remove existing object, if it exists
  GeoObject old;
  Status status = GetById(obj.id, &old);
  if (status.ok()) {
    assert(obj.id.compare(old.id) == 0);
    std::string quadkey = PositionToQuad(old.position, Detail);
    std::string key1 = MakeKey1(old.position, old.id, quadkey);
    std::string key2 = MakeKey2(old.id);
    batch.Delete(Slice(key1));
    batch.Delete(Slice(key2));
  } else if (status.IsNotFound()) {
    // What if another thread is trying to insert the same ID concurrently?
  } else {
    return status;
  }

  // insert new object
  std::string quadkey = PositionToQuad(obj.position, Detail);
  std::string key1 = MakeKey1(obj.position, obj.id, quadkey);
  std::string key2 = MakeKey2(obj.id);
  batch.Put(Slice(key1), Slice(obj.value));
  batch.Put(Slice(key2), Slice(quadkey));
  return db_->Write(woptions_, &batch);
}

Status GeoDBImpl::GetByPosition(const GeoPosition& pos,
                                const Slice& id,
                                std::string* value) {
  std::string quadkey = PositionToQuad(pos, Detail);
  std::string key1 = MakeKey1(pos, id, quadkey);
  return db_->Get(roptions_, Slice(key1), value);
}

Status GeoDBImpl::GetById(const Slice& id, GeoObject* object) {
  Status status;
  std::string quadkey;

  // create an iterator so that we can get a consistent picture
  // of the database.
  Iterator* iter = db_->NewIterator(roptions_);

  // create key for table2
  std::string kt = MakeKey2(id);
  Slice key2(kt);

  iter->Seek(key2);
  if (iter->Valid() && iter->status().ok()) {
    if (iter->key().compare(key2) == 0) {
      quadkey = iter->value().ToString();
    }
  }
  if (quadkey.size() == 0) {
    delete iter;
    return Status::NotFound(key2);
  }

  //
  // Seek to the quadkey + id prefix
  //
  std::string prefix = MakeKey1Prefix(quadkey, id);
  iter->Seek(Slice(prefix));
  assert(iter->Valid());
  if (!iter->Valid() || !iter->status().ok()) {
    delete iter;
    return Status::NotFound();
  }

  // split the key into p + quadkey + id + lat + lon
  Slice key = iter->key();
  std::vector<std::string> parts = StringSplit(key.ToString(), ':');
  assert(parts.size() == 5);
  assert(parts[0] == "p");
  assert(parts[1] == quadkey);
  assert(parts[2] == id);

  // fill up output parameters
  object->position.latitude = atof(parts[3].c_str());
  object->position.longitude = atof(parts[4].c_str());
  object->id = id.ToString();  // this is redundant
  object->value = iter->value().ToString();
  delete iter;
  return Status::OK();
}


Status GeoDBImpl::Remove(const Slice& id) {
  // Read the object from the database
  GeoObject obj;
  Status status = GetById(id, &obj);
  if (!status.ok()) {
    return status;
  }

  // remove the object by atomically deleting it from both tables
  std::string quadkey = PositionToQuad(obj.position, Detail);
  std::string key1 = MakeKey1(obj.position, obj.id, quadkey);
  std::string key2 = MakeKey2(obj.id);
  WriteBatch batch;
  batch.Delete(Slice(key1));
  batch.Delete(Slice(key2));
  return db_->Write(woptions_, &batch);
}

Status GeoDBImpl::SearchRadial(const GeoPosition& pos,
  double radius,
  std::vector<GeoObject>* values,
  int number_of_values) {
  // Gather all bounding quadkeys
  std::vector<std::string> qids;
  Status s = searchQuadIds(pos, radius, &qids);
  if (!s.ok()) {
    return s;
  }

  // create an iterator
  Iterator* iter = db_->NewIterator(ReadOptions());

  // Process each prospective quadkey
  for (std::string qid : qids) {
    // The user is interested in only these many objects.
    if (number_of_values == 0) {
      break;
    }

    // convert quadkey to db key prefix
    std::string dbkey = MakeQuadKeyPrefix(qid);

    for (iter->Seek(dbkey);
         number_of_values > 0 && iter->Valid() && iter->status().ok();
         iter->Next()) {
      // split the key into p + quadkey + id + lat + lon
      Slice key = iter->key();
      std::vector<std::string> parts = StringSplit(key.ToString(), ':');
      assert(parts.size() == 5);
      assert(parts[0] == "p");
      std::string* quadkey = &parts[1];

      // If the key we are looking for is a prefix of the key
      // we found from the database, then this is one of the keys
      // we are looking for.
      auto res = std::mismatch(qid.begin(), qid.end(), quadkey->begin());
      if (res.first == qid.end()) {
        GeoPosition obj_pos(atof(parts[3].c_str()), atof(parts[4].c_str()));
        GeoObject obj(obj_pos, parts[4], iter->value().ToString());
        values->push_back(obj);
        number_of_values--;
      } else {
        break;
      }
    }
  }
  delete iter;
  return Status::OK();
}

std::string GeoDBImpl::MakeKey1(const GeoPosition& pos, Slice id,
                                std::string quadkey) {
  std::string lat = rocksdb::ToString(pos.latitude);
  std::string lon = rocksdb::ToString(pos.longitude);
  std::string key = "p:";
  key.reserve(5 + quadkey.size() + id.size() + lat.size() + lon.size());
  key.append(quadkey);
  key.append(":");
  key.append(id.ToString());
  key.append(":");
  key.append(lat);
  key.append(":");
  key.append(lon);
  return key;
}

std::string GeoDBImpl::MakeKey2(Slice id) {
  std::string key = "k:";
  key.append(id.ToString());
  return key;
}

std::string GeoDBImpl::MakeKey1Prefix(std::string quadkey,
                                      Slice id) {
  std::string key = "p:";
  key.reserve(3 + quadkey.size() + id.size());
  key.append(quadkey);
  key.append(":");
  key.append(id.ToString());
  return key;
}

std::string GeoDBImpl::MakeQuadKeyPrefix(std::string quadkey) {
  std::string key = "p:";
  key.append(quadkey);
  return key;
}

// convert degrees to radians
double GeoDBImpl::radians(double x) {
  return (x * PI) / 180;
}

// convert radians to degrees
double GeoDBImpl::degrees(double x) {
  return (x * 180) / PI;
}

// convert a gps location to quad coordinate
std::string GeoDBImpl::PositionToQuad(const GeoPosition& pos,
                                      int levelOfDetail) {
  Pixel p = PositionToPixel(pos, levelOfDetail);
  Tile tile = PixelToTile(p);
  return TileToQuadKey(tile, levelOfDetail);
}

GeoPosition GeoDBImpl::displaceLatLon(double lat, double lon,
                                      double deltay, double deltax) {
  double dLat = deltay / EarthRadius;
  double dLon = deltax / (EarthRadius * cos(radians(lat)));
  return GeoPosition(lat + degrees(dLat),
                     lon + degrees(dLon));
}

//
// Return the distance between two positions on the earth
//
double GeoDBImpl::distance(double lat1, double lon1,
                           double lat2, double lon2) {
  double lon = radians(lon2 - lon1);
  double lat = radians(lat2 - lat1);

  double a = (sin(lat / 2) * sin(lat / 2)) +
              cos(radians(lat1)) * cos(radians(lat2)) *
              (sin(lon / 2) * sin(lon / 2));
  double angle = 2 * atan2(sqrt(a), sqrt(1 - a));
  return angle * EarthRadius;
}

//
// Returns all the quadkeys inside the search range
//
Status GeoDBImpl::searchQuadIds(const GeoPosition& position,
                                double radius,
                                std::vector<std::string>* quadKeys) {
  // get the outline of the search square
  GeoPosition topLeftPos = boundingTopLeft(position, radius);
  GeoPosition bottomRightPos = boundingBottomRight(position, radius);

  Pixel topLeft =  PositionToPixel(topLeftPos, Detail);
  Pixel bottomRight =  PositionToPixel(bottomRightPos, Detail);

  // how many level of details to look for
  int numberOfTilesAtMaxDepth = floor((bottomRight.x - topLeft.x) / 256);
  int zoomLevelsToRise = floor(::log(numberOfTilesAtMaxDepth) / ::log(2));
  zoomLevelsToRise++;
  int levels = std::max(0, Detail - zoomLevelsToRise);

  quadKeys->push_back(PositionToQuad(GeoPosition(topLeftPos.latitude,
                                                 topLeftPos.longitude),
                                     levels));
  quadKeys->push_back(PositionToQuad(GeoPosition(topLeftPos.latitude,
                                                 bottomRightPos.longitude),
                                     levels));
  quadKeys->push_back(PositionToQuad(GeoPosition(bottomRightPos.latitude,
                                                 topLeftPos.longitude),
                                     levels));
  quadKeys->push_back(PositionToQuad(GeoPosition(bottomRightPos.latitude,
                                                 bottomRightPos.longitude),
                                     levels));
  return Status::OK();
}

// Determines the ground resolution (in meters per pixel) at a specified
// latitude and level of detail.
// Latitude (in degrees) at which to measure the ground resolution.
// Level of detail, from 1 (lowest detail) to 23 (highest detail).
// Returns the ground resolution, in meters per pixel.
double GeoDBImpl::GroundResolution(double latitude, int levelOfDetail) {
  latitude = clip(latitude, MinLatitude, MaxLatitude);
  return cos(latitude * PI / 180) * 2 * PI * EarthRadius /
         MapSize(levelOfDetail);
}

// Converts a point from latitude/longitude WGS-84 coordinates (in degrees)
// into pixel XY coordinates at a specified level of detail.
GeoDBImpl::Pixel GeoDBImpl::PositionToPixel(const GeoPosition& pos,
                                            int levelOfDetail) {
  double latitude = clip(pos.latitude, MinLatitude, MaxLatitude);
  double x = (pos.longitude + 180) / 360;
  double sinLatitude = sin(latitude * PI / 180);
  double y = 0.5 - ::log((1 + sinLatitude) / (1 - sinLatitude)) / (4 * PI);
  double mapSize = MapSize(levelOfDetail);
  double X = floor(clip(x * mapSize + 0.5, 0, mapSize - 1));
  double Y = floor(clip(y * mapSize + 0.5, 0, mapSize - 1));
  return Pixel((unsigned int)X, (unsigned int)Y);
}

GeoPosition GeoDBImpl::PixelToPosition(const Pixel& pixel, int levelOfDetail) {
  double mapSize = MapSize(levelOfDetail);
  double x = (clip(pixel.x, 0, mapSize - 1) / mapSize) - 0.5;
  double y = 0.5 - (clip(pixel.y, 0, mapSize - 1) / mapSize);
  double latitude = 90 - 360 * atan(exp(-y * 2 * PI)) / PI;
  double longitude = 360 * x;
  return GeoPosition(latitude, longitude);
}

// Converts a Pixel to a Tile
GeoDBImpl::Tile GeoDBImpl::PixelToTile(const Pixel& pixel) {
  unsigned int tileX = floor(pixel.x / 256);
  unsigned int tileY = floor(pixel.y / 256);
  return Tile(tileX, tileY);
}

GeoDBImpl::Pixel GeoDBImpl::TileToPixel(const Tile& tile) {
  unsigned int pixelX = tile.x * 256;
  unsigned int pixelY = tile.y * 256;
  return Pixel(pixelX, pixelY);
}

// Convert a Tile to a quadkey
std::string GeoDBImpl::TileToQuadKey(const Tile& tile, int levelOfDetail) {
  std::stringstream quadKey;
  for (int i = levelOfDetail; i > 0; i--) {
    char digit = '0';
    int mask = 1 << (i - 1);
    if ((tile.x & mask) != 0) {
      digit++;
    }
    if ((tile.y & mask) != 0) {
      digit++;
      digit++;
    }
    quadKey << digit;
  }
  return quadKey.str();
}

//
// Convert a quadkey to a tile and its level of detail
//
void GeoDBImpl::QuadKeyToTile(std::string quadkey, Tile* tile,
                              int* levelOfDetail) {
  tile->x = tile->y = 0;
  *levelOfDetail = static_cast<int>(quadkey.size());
  const char* key = reinterpret_cast<const char*>(quadkey.c_str());
  for (int i = *levelOfDetail; i > 0; i--) {
    int mask = 1 << (i - 1);
    switch (key[*levelOfDetail - i]) {
      case '0':
        break;

      case '1':
        tile->x |= mask;
        break;

      case '2':
        tile->y |= mask;
        break;

      case '3':
        tile->x |= mask;
        tile->y |= mask;
        break;

      default:
        std::stringstream msg;
        msg << quadkey;
        msg << " Invalid QuadKey.";
        throw std::runtime_error(msg.str());
    }
  }
}
}  // namespace rocksdb

#endif  // ROCKSDB_LITE
